(function () {
  "use strict";

  const S = function (sel) { return document.querySelector(sel); };

  var loginSection = S("#login-section");
  var dashSection = S("#dashboard-section");
  var loginForm = S("#login-form");
  var loginError = S("#login-error");
  var configForm = S("#config-form");
  var configStatus = S("#config-status");
  var sessionsBody = S("#sessions-table tbody");
  var connectionsBody = S("#connections-table tbody");
  var usageSummary = S("#usage-summary");
  var usageFill = S("#usage-fill");

  var auth = { user: "", pass: "" };
  var refreshTimer = null;

  function apiFetch(url, opts) {
    opts = opts || {};
    opts.headers = opts.headers || {};
    opts.headers["Authorization"] = "Basic " + btoa(auth.user + ":" + auth.pass);
    return fetch(url, opts).then(function (res) {
      if (res.status === 401) {
        auth = { user: "", pass: "" };
        showLogin();
        throw new Error("Authentication required");
      }
      return res;
    });
  }

  function apiJson(url, opts) {
    return apiFetch(url, opts).then(function (res) {
      if (!res.ok) throw new Error(res.status + " " + res.statusText);
      return res.json();
    });
  }

  function showLogin() {
    loginSection.classList.remove("hidden");
    dashSection.classList.add("hidden");
    loginError.innerHTML = "";
  }

  function showDashboard() {
    loginSection.classList.add("hidden");
    dashSection.classList.remove("hidden");
  }

  loginForm.addEventListener("submit", function (e) {
    e.preventDefault();
    auth.user = S("#user").value;
    auth.pass = S("#pass").value;

    apiJson("/api/admin/sessions")
      .then(function () {
        showDashboard();
        loadDashboard();
      })
      .catch(function () {
        loginError.innerHTML = '<div class="error">Invalid credentials</div>';
        auth = { user: "", pass: "" };
      });
  });

  S("#logout-btn").addEventListener("click", function () {
    auth = { user: "", pass: "" };
    S("#user").value = "";
    S("#pass").value = "";
    clearInterval(refreshTimer);
    refreshTimer = null;
    showLogin();
  });

  function loadDashboard() {
    Promise.all([
      apiJson("/api/admin/sessions"),
      apiJson("/api/admin/config"),
      apiJson("/api/admin/usage"),
    ])
      .then(function (results) {
        renderSessions(results[0]);
        renderConfig(results[1]);
        renderUsage(results[2]);
      })
      .catch(function (err) {
        loginError.innerHTML = '<div class="error">' + err.message + "</div>";
      });

    refreshConnections();
    refreshTimer = setInterval(refreshConnections, 5000);
  }

  function renderConfig(cfg) {
    S("#cfg-chunk").value = cfg.chunkDurationMs;
    S("#cfg-bitrate").value = cfg.videoBitrate;
    S("#cfg-maxw").value = cfg.maxWidth;
    S("#cfg-maxh").value = cfg.maxHeight;
    S("#cfg-storage").value = cfg.storagePath;
    S("#cfg-adminuser").value = cfg.adminUser;
    S("#cfg-adminpass").value = "";
    S("#cfg-fields").value = (cfg.infoFields || []).join("\n");
    S("#cfg-audio").checked = cfg.audioEnabled !== false;
  }

  configForm.addEventListener("submit", function (e) {
    e.preventDefault();
    var fd = new FormData(configForm);
    var cfg = {};

    fd.forEach(function (v, k) {
      if (k === "infoFields") {
        cfg[k] = String(v)
          .split("\n")
          .map(function (s) { return s.trim(); })
          .filter(Boolean);
      } else if (k === "chunkDurationMs" || k === "videoBitrate" || k === "maxWidth" || k === "maxHeight") {
        cfg[k] = parseInt(v, 10) || 0;
      } else if (k === "audioEnabled") {
        cfg[k] = true;
      } else {
        cfg[k] = k === "adminPass" && v === "" ? "" : String(v);
      }
    });

    if (cfg.audioEnabled === undefined) cfg.audioEnabled = false;

    apiFetch("/api/admin/config", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(cfg),
    })
      .then(function () {
        configStatus.innerHTML = '<div class="banner">Configuration saved.</div>';
        setTimeout(function () { configStatus.innerHTML = ""; }, 3000);
        if (cfg.adminPass) {
          auth.pass = cfg.adminPass;
        }
      })
      .catch(function (err) {
        configStatus.innerHTML = '<div class="error">Failed to save: ' + err.message + "</div>";
      });
  });

  function renderSessions(sessions) {
    if (!sessions.length) {
      sessionsBody.innerHTML =
        '<tr><td colspan="6" style="text-align:center;color:#8b7355;">No sessions yet.</td></tr>';
      return;
    }

    var rows = "";
    for (var i = 0; i < sessions.length; i++) {
      var s = sessions[i];
      var takes = s.takes || [];
      var totalSize = 0;
      for (var j = 0; j < takes.length; j++) {
        totalSize += takes[j].bytesWritten || 0;
      }

      var takeList = "";
      for (var j = 0; j < takes.length; j++) {
        var t = takes[j];
        if (j > 0) takeList += ", ";
        takeList += t.file + ' <span style="color:#8b7355;">(' + (t.completed ? "done" : "partial") + ")</span>";
      }

      var actions = "";
      for (var j = 0; j < takes.length; j++) {
        var t = takes[j];
        if (j > 0) actions += " ";
        actions +=
          '<a href="/api/admin/sessions/file?id=' +
          encodeURIComponent(s.uuid) +
          "&file=" +
          encodeURIComponent(t.file) +
          '">' +
          t.file +
          "</a>";
      }
      actions +=
        ' <a href="/api/admin/sessions/zip?id=' +
        encodeURIComponent(s.uuid) +
        '">zip</a>';

      rows +=
        "<tr>" +
        "<td>" + (s.name || "-") + "</td>" +
        "<td><code>" +
        s.uuid +
        "</code></td>" +
        "<td>" +
        formatDate(s.createdAt) +
        "</td>" +
        "<td>" +
        (takeList || "-") +
        "</td>" +
        "<td>" +
        formatBytes(totalSize) +
        "</td>" +
        '<td class="actions">' +
        actions +
        "</td>" +
        "</tr>";
    }
    sessionsBody.innerHTML = rows;
  }

  function renderUsage(usage) {
    usageSummary.textContent = "Storage: " + formatBytes(usage.bytes);
    usageFill.style.width = "100%";
  }

  function refreshConnections() {
    apiJson("/api/admin/connections")
      .then(renderConnections)
      .catch(function () {});
  }

  function renderConnections(conns) {
    if (!conns.length) {
      connectionsBody.innerHTML =
        '<tr><td colspan="6" style="text-align:center;color:#8b7355;">No active connections.</td></tr>';
      return;
    }
    var rows = "";
    for (var i = 0; i < conns.length; i++) {
      var c = conns[i];
      rows +=
        "<tr>" +
        "<td>" + (c.name || "-") + "</td>" +
        "<td><code>" + c.sessionUUID + "</code></td>" +
        "<td>" + c.take + "</td>" +
        "<td>" + (c.clientIP || "-") + "</td>" +
        "<td>" + formatDate(c.connectedAt) + "</td>" +
        "<td>" + formatBytes(c.bytesSent) + "</td>" +
        "</tr>";
    }
    connectionsBody.innerHTML = rows;
  }

  function formatDate(iso) {
    if (!iso) return "-";
    try {
      return new Date(iso).toLocaleString();
    } catch (e) {
      return iso;
    }
  }

  function formatBytes(bytes) {
    if (!bytes) return "0 B";
    var units = ["B", "KB", "MB", "GB"];
    var i = 0;
    var n = bytes;
    while (n >= 1024 && i < units.length - 1) {
      n /= 1024;
      i++;
    }
    return n.toFixed(i > 0 ? 1 : 0) + " " + units[i];
  }

  showLogin();
})();
