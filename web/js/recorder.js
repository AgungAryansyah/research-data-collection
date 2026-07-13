(function () {
  "use strict";

  const STORAGE_KEY = "research-session";

  const $ = (s) => document.querySelector(s);
  const formSection = $("#form-section");
  const cameraSection = $("#camera-section");
  const doneSection = $("#done-section");
  const infoFields = $("#info-fields");
  const infoForm = $("#info-form");
  const preview = $("#preview");
  const cameraMeta = $("#camera-meta");
  const startBtn = $("#start-recording");
  const stopBtn = $("#stop-recording");
  const timerEl = $("#recording-timer");
  const statusEl = $("#upload-status");
  const sessionIdEl = $("#session-id");

  let state = {
    config: null,
    session: null,
    stream: null,
    metadata: null,
    mediaRecorder: null,
    ws: null,
    timer: null,
    seconds: 0,
  };

  async function init() {
    try {
      state.config = await fetchConfig();
      state.session = loadSession();

      if (state.session) {
        showCamera();
      } else {
        showForm();
      }
    } catch (err) {
      flashError("Failed to initialize: " + err.message);
    }
  }

  async function fetchConfig() {
    const res = await fetch("/api/config");
    if (!res.ok) throw new Error("config fetch failed");
    return res.json();
  }

  function loadSession() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  }

  function saveSession() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state.session));
  }

  function clearSession() {
    localStorage.removeItem(STORAGE_KEY);
    state.session = null;
  }

  function showForm() {
    formSection.classList.remove("hidden");
    cameraSection.classList.add("hidden");
    doneSection.classList.add("hidden");
    buildFormFields();
    clearBanner();
  }

  function buildFormFields() {
    infoFields.innerHTML = "";
    for (const field of state.config.infoFields) {
      const div = document.createElement("div");
      div.className = "field";
      const label = document.createElement("label");
      label.textContent = field;
      const input = document.createElement("input");
      input.type = "text";
      input.name = field;
      input.id = "field-" + field.toLowerCase().replace(/\s+/g, "-");
      input.required = true;

      if (state.session && state.session.info && state.session.info[field]) {
        input.value = state.session.info[field];
      }

      div.appendChild(label);
      div.appendChild(input);
      infoFields.appendChild(div);
    }
  }

  infoForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(infoForm);
    const info = {};
    for (const [k, v] of fd.entries()) info[k] = String(v);

    const uuid = state.session ? state.session.uuid : crypto.randomUUID();
    state.session = { uuid, info, nextTake: 1 };
    saveSession();
    await showCamera();
  });

  async function showCamera() {
    formSection.classList.add("hidden");
    cameraSection.classList.remove("hidden");
    doneSection.classList.add("hidden");
    clearBanner();

    if (state.session.nextTake > 1) {
      showBanner("Previous recording may have been interrupted. Starting a new take.");
    }

    try {
      state.stream = await navigator.mediaDevices.getUserMedia({
        video: { width: { ideal: state.config.maxWidth }, height: { ideal: state.config.maxHeight } },
        audio: state.config.audioEnabled !== false,
      });
      preview.srcObject = state.stream;
      await detectMetadata();
      startBtn.disabled = false;
      stopBtn.disabled = true;
    } catch (err) {
      if (err.name === "NotAllowedError" || err.name === "NotFoundError") {
        flashError("Camera access denied or not available. Please check your camera permissions.");
      } else {
        flashError("Camera error: " + err.message);
      }
    }
  }

  async function detectMetadata() {
    const track = state.stream.getVideoTracks()[0];
    const settings = track.getSettings();
    const devices = await navigator.mediaDevices.enumerateDevices();
    const vd = devices.find((d) => d.deviceId === settings.deviceId);

    state.metadata = {
      camera: {
        label: vd ? vd.label : "unknown",
        deviceId: settings.deviceId || "unknown",
        groupId: vd ? vd.groupId : "unknown",
        facingMode: settings.facingMode || "unknown",
      },
      resolution: { width: settings.width, height: settings.height },
      frameRate: settings.frameRate,
      aspectRatio: settings.aspectRatio || 0,
      resizeMode: settings.resizeMode || "none",
      userAgent: navigator.userAgent,
      platform: navigator.platform,
      collectedAt: new Date().toISOString(),
    };

    cameraMeta.innerHTML =
      "<p>Camera: " +
      state.metadata.camera.label +
      "</p>" +
      "<p>Resolution: " +
      state.metadata.resolution.width +
      "x" +
      state.metadata.resolution.height +
      "</p>" +
      "<p>Frame rate: " +
      state.metadata.frameRate +
      " fps</p>";
  }

  startBtn.addEventListener("click", startRecording);
  stopBtn.addEventListener("click", stopRecording);

  async function startRecording() {
    const take = state.session.nextTake;
    state.session.nextTake = take + 1;
    saveSession();

    const protocol = location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = protocol + "//" + location.host + "/ws/upload?session=" + encodeURIComponent(state.session.uuid) + "&take=" + take;

    try {
      state.ws = new WebSocket(wsUrl);
      state.ws.binaryType = "blob";

      await new Promise((resolve, reject) => {
        state.ws.onopen = resolve;
        state.ws.onerror = () => reject(new Error("WebSocket connection failed"));
      });

      state.ws.send(JSON.stringify({ info: state.session.info, metadata: state.metadata }));

      state.mediaRecorder = new MediaRecorder(state.stream, {
        mimeType: getMimeType(),
        videoBitsPerSecond: state.config.videoBitrate,
      });

      state.mediaRecorder.ondataavailable = (ev) => {
        if (ev.data.size > 0 && state.ws && state.ws.readyState === WebSocket.OPEN) {
          state.ws.send(ev.data);
        }
      };

      state.mediaRecorder.onstop = () => {
        if (state.ws && state.ws.readyState === WebSocket.OPEN) {
          state.ws.send(JSON.stringify({ type: "finalize" }));
        }
      };

      state.ws.onclose = () => {
        state.ws = null;
        if (!state.finalized) showDone();
      };

      state.ws.onerror = () => {
        flashError("Upload connection lost. The recording may be partial.");
        cleanupRecording();
      };

      state.mediaRecorder.start(state.config.chunkDurationMs);

      startBtn.disabled = true;
      stopBtn.disabled = false;
      state.seconds = 0;
      updateTimer();
      state.timer = setInterval(updateTimer, 1000);
      statusEl.textContent = "Recording...";
    } catch (err) {
      flashError("Recording failed: " + err.message);
      if (state.ws) {
        state.ws.close();
        state.ws = null;
      }
      startBtn.disabled = false;
    }
  }

  function updateTimer() {
    const m = String(Math.floor(state.seconds / 60)).padStart(2, "0");
    const s = String(state.seconds % 60).padStart(2, "0");
    timerEl.textContent = m + ":" + s;
    state.seconds++;
  }

  function stopRecording() {
    clearInterval(state.timer);
    state.timer = null;
    statusEl.textContent = "Finalizing...";
    state.finalized = true;

    if (state.mediaRecorder && state.mediaRecorder.state === "recording") {
      state.mediaRecorder.stop();
    }

    Promise.race([
      new Promise((resolve) => {
        state.ws.onclose = () => {
          state.ws = null;
          resolve();
        };
      }),
      new Promise((resolve) => setTimeout(resolve, 5000)),
    ]).then(() => {
      cleanupRecording();
      showDone();
    });
  }

  function cleanupRecording() {
    clearInterval(state.timer);
    state.timer = null;
    if (state.ws) {
      state.ws.close();
      state.ws = null;
    }
    if (state.mediaRecorder && state.mediaRecorder.state === "recording") {
      state.mediaRecorder.stop();
    }
    state.mediaRecorder = null;
    startBtn.disabled = false;
    stopBtn.disabled = true;
  }

  function stopStream() {
    if (state.stream) {
      state.stream.getTracks().forEach((t) => t.stop());
      state.stream = null;
    }
  }

  function showDone() {
    formSection.classList.add("hidden");
    cameraSection.classList.add("hidden");
    doneSection.classList.remove("hidden");
    sessionIdEl.textContent = state.session.uuid;
    stopStream();
    statusEl.textContent = "";
    timerEl.textContent = "00:00";
  }

  function showBanner(msg) {
    let el = $(".banner");
    if (!el) {
      el = document.createElement("div");
      el.className = "banner";
      cameraSection.insertBefore(el, cameraSection.firstChild);
    }
    el.textContent = msg;
  }

  function clearBanner() {
    const el = $(".banner");
    if (el) el.remove();
  }

  function flashError(msg) {
    const el = document.createElement("div");
    el.className = "error";
    el.textContent = msg;
    document.querySelector(".container").prepend(el);
    setTimeout(() => el.remove(), 6000);
  }

  function getMimeType() {
    const types = ["video/webm;codecs=vp9,opus", "video/webm;codecs=vp8,opus", "video/webm"];
    for (const t of types) {
      if (MediaRecorder.isTypeSupported(t)) return t;
    }
    throw new Error("MediaRecorder is not supported in this browser");
  }

  window.addEventListener("beforeunload", (e) => {
    if (state.mediaRecorder && state.mediaRecorder.state === "recording") {
      e.preventDefault();
      e.returnValue = "";
    }
  });

  $("#cancel-session").addEventListener("click", () => {
    clearSession();
    stopStream();
    cleanupRecording();
    showForm();
  });

  $("#new-session-btn").addEventListener("click", () => {
    clearSession();
    stopStream();
    showForm();
  });

  init();
})();
