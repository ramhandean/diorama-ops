(function () {
  'use strict';

  try {
    if (navigator.doNotTrack === '1' || window.doNotTrack === '1') return;

    var script = document.currentScript || document.querySelector('script[data-key]');
    if (!script) return;

    var siteKey = script.getAttribute('data-key');
    if (!siteKey) return;

    var src = script.getAttribute('src') || '';
    var hubHost = src.indexOf('http') === 0 ? new URL(src).host : window.location.host;
    var wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    var httpProto = window.location.protocol === 'https:' ? 'https:' : 'http:';

    var sid = 's_' + Math.random().toString(36).substring(2, 10) + Date.now().toString(36).slice(-4);
    var ws = null;
    var hbTimer = null;
    var retryDelay = 2000;
    var useBeaconFallback = false;

    function getPath() {
      return window.location.pathname || '/';
    }

    function getRef() {
      try {
        if (!document.referrer) return '';
        var refUrl = new URL(document.referrer);
        if (refUrl.host === window.location.host) return '';
        return refUrl.host;
      } catch (e) {
        return '';
      }
    }

    function sendBeacon(msg) {
      try {
        var url = httpProto + '//' + hubHost + '/api/v1/b';
        var data = JSON.stringify(msg);
        if (navigator.sendBeacon) {
          navigator.sendBeacon(url, data);
        } else {
          fetch(url, {
            method: 'POST',
            body: data,
            headers: { 'Content-Type': 'application/json' },
            keepalive: true
          }).catch(function () {});
        }
      } catch (e) {}
    }

    function sendMsg(msg) {
      if (ws && ws.readyState === WebSocket.OPEN) {
        try {
          ws.send(JSON.stringify(msg));
          return;
        } catch (e) {}
      }
      sendBeacon(msg);
    }

    function sendHeartbeat() {
      if (document.visibilityState === 'hidden') return;
      sendMsg({ t: 'hb', sid: sid, path: getPath() });
    }

    function startHeartbeat() {
      stopHeartbeat();
      hbTimer = setInterval(sendHeartbeat, 15000);
    }

    function stopHeartbeat() {
      if (hbTimer) {
        clearInterval(hbTimer);
        hbTimer = null;
      }
    }

    function connectWS() {
      if (useBeaconFallback) return;

      try {
        var wsUrl = wsProto + '//' + hubHost + '/ws';
        ws = new WebSocket(wsUrl);

        ws.onopen = function () {
          retryDelay = 2000;
          ws.send(JSON.stringify({
            t: 'hello',
            key: siteKey,
            sid: sid,
            path: getPath(),
            ref: getRef()
          }));
          startHeartbeat();
        };

        ws.onclose = function () {
          stopHeartbeat();
          // Exponential backoff with jitter up to 30s
          retryDelay = Math.min(retryDelay * 1.5 + Math.random() * 1000, 30000);
          setTimeout(connectWS, retryDelay);
        };

        ws.onerror = function () {
          try { ws.close(); } catch (e) {}
        };
      } catch (e) {
        useBeaconFallback = true;
        sendBeacon({
          t: 'hello',
          key: siteKey,
          sid: sid,
          path: getPath(),
          ref: getRef()
        });
        startHeartbeat();
      }
    }

    // Connect
    connectWS();

    // SPA navigation observation
    var lastPath = getPath();
    function checkNav() {
      var currentPath = getPath();
      if (currentPath !== lastPath) {
        lastPath = currentPath;
        sendMsg({ t: 'nav', sid: sid, path: currentPath });
      }
    }

    var originalPush = history.pushState;
    if (originalPush) {
      history.pushState = function () {
        originalPush.apply(this, arguments);
        checkNav();
      };
    }

    var originalReplace = history.replaceState;
    if (originalReplace) {
      history.replaceState = function () {
        originalReplace.apply(this, arguments);
        checkNav();
      };
    }

    window.addEventListener('popstate', checkNav, { passive: true });

    // Tab visibility handling
    document.addEventListener('visibilitychange', function () {
      if (document.visibilityState === 'visible') {
        sendHeartbeat();
        startHeartbeat();
      } else {
        stopHeartbeat();
      }
    }, { passive: true });

    // Session termination
    var exited = false;
    function onExit() {
      if (exited) return;
      exited = true;
      stopHeartbeat();
      sendBeacon({ t: 'bye', sid: sid });
    }

    window.addEventListener('pagehide', onExit, { passive: true });
    window.addEventListener('beforeunload', onExit, { passive: true });

  } catch (err) {}
})();
