/* Funciones compartidas por todas las pantallas.
   Escrito en JavaScript "viejo" a propósito, para que ande también
   en el navegador del Chromecast y en Chrome 109 (Windows 7). */
(function (w) {
  'use strict';

  var SOC = /^(S\.?\s?A\.?|S\.?\s?R\.?\s?L\.?|S\.?\s?A\.?\s?S\.?|SACIF|SACIFIA|SAIC|SAICA|SACI|SAICF|S\.?\s?C\.?\s?A\.?|S\.?\s?H\.?|S\.?\s?C\.?|SOC\.?.*|COOP.*|LTDA\.?)$/i;
  var SOC_EN = /\b(S\.?A\.?|S\.?R\.?L\.?|S\.?A\.?S\.?|SACIF|SAIC|SACI|HNOS\.?|COOP\w*|LTDA|SOCIEDAD)\b/i;

  function esSociedad(n) {
    var p = n.split(',');
    return SOC.test((p[0] || '').trim()) || (SOC_EN.test(n) && /S\.A\.|S\.R\.L\.|HNOS|SRL|\bSA\b/i.test(n));
  }

  // "SA, AGRICOLA" -> "AGRICOLA SA"; "WALLACE, HNOS S.A." -> "WALLACE HNOS S.A."; personas quedan "APELLIDO, NOMBRE"
  function nombre(n) {
    n = String(n || '').replace(/\s+/g, ' ').trim();
    var p = n.split(',');
    if (p.length === 2) {
      var a = p[0].trim(), b = p[1].trim();
      if (SOC.test(a)) return b + ' ' + a;
      if (esSociedad(n)) return a + ' ' + b;
      return a + ', ' + b;
    }
    return n;
  }

  function esc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }

  function hhmm(iso) {
    if (!iso || iso.indexOf('0001-') === 0) return '';
    var d = new Date(iso);
    if (isNaN(d.getTime())) return '';
    return ('0' + d.getHours()).slice(-2) + ':' + ('0' + d.getMinutes()).slice(-2);
  }

  function minutosDesde(iso, ahora) {
    if (!iso || iso.indexOf('0001-') === 0) return null;
    var m = Math.round(((ahora ? new Date(ahora) : new Date()) - new Date(iso)) / 60000);
    return m < 0 ? 0 : m;
  }

  // ---- Sonidos de llamado (se generan en el momento, no hay archivos de audio) ----
  var ctx = null;
  function audio() {
    try {
      if (!ctx) {
        var C = w.AudioContext || w.webkitAudioContext;
        if (!C) return null;
        ctx = new C();
      }
      if (ctx.state === 'suspended' && ctx.resume) ctx.resume();
      return ctx;
    } catch (e) { return null; }
  }
  function nota(c, t, f, dur, tipo, vol, arm) {
    arm = arm || [[1, 1]];
    for (var i = 0; i < arm.length; i++) {
      var o = c.createOscillator(), g = c.createGain();
      o.type = tipo || 'sine';
      o.frequency.value = f * arm[i][0];
      o.connect(g); g.connect(c.destination);
      var v = (vol || 0.4) * arm[i][1];
      g.gain.setValueAtTime(0.0001, t);
      g.gain.exponentialRampToValueAtTime(v, t + 0.015);
      g.gain.exponentialRampToValueAtTime(0.0001, t + dur);
      o.start(t); o.stop(t + dur + 0.05);
    }
  }
  var SONIDOS = [
    ['dingdong', 'Ding-dong clásico', 'Dos notas que bajan, como un timbre. Es el más conocido y no molesta.'],
    ['campana', 'Campanita', 'Una sola campanada limpia que se apaga despacio. Muy discreto.'],
    ['tres', 'Tres tonos', 'Tres notas que suben, estilo aeropuerto. Se escucha bien con ruido de sala.'],
    ['marimba', 'Marimba', 'Tres golpecitos cortos y alegres. Moderno y suave.'],
    ['digital', 'Aviso digital', 'Doble bip corto, estilo turnero de banco. Seco y directo.'],
    ['carillon', 'Carillón', 'Cuatro notas tipo reloj de campanario. El más largo y llamativo.']
  ];
  function tocar(id) {
    var c = audio();
    if (!c) return 0;
    var t = c.currentTime + 0.05, dur = 1.6;
    var camp = [[1, 1], [2.76, 0.35], [5.4, 0.15]], mar = [[1, 1], [4, 0.25]];
    if (id === 'campana') { nota(c, t, 880, 2.2, 'sine', 0.35, camp); dur = 2.3; }
    else if (id === 'tres') { nota(c, t, 523, 0.9, 'sine', 0.35); nota(c, t + 0.28, 659, 0.9, 'sine', 0.35); nota(c, t + 0.56, 784, 1.4, 'sine', 0.35); dur = 2; }
    else if (id === 'marimba') { nota(c, t, 784, 0.5, 'sine', 0.45, mar); nota(c, t + 0.16, 988, 0.5, 'sine', 0.45, mar); nota(c, t + 0.32, 1175, 0.9, 'sine', 0.45, mar); dur = 1.3; }
    else if (id === 'digital') { nota(c, t, 1320, 0.16, 'square', 0.12); nota(c, t + 0.3, 1320, 0.16, 'square', 0.12); dur = 0.8; }
    else if (id === 'carillon') { var ns = [784, 659, 698, 523]; for (var j = 0; j < 4; j++) nota(c, t + j * 0.42, ns[j], 1.3, 'sine', 0.35, [[1, 1], [2, 0.25], [3, 0.08]]); dur = 3; }
    else { nota(c, t, 659, 1.3, 'sine', 0.45, [[1, 1], [2, 0.2]]); nota(c, t + 0.45, 523, 1.6, 'sine', 0.45, [[1, 1], [2, 0.2]]); dur = 2.1; }
    return dur;
  }
  function audioActivo() { return !!(ctx && ctx.state === 'running'); }

  // ---- Conexión en tiempo real con la PC central ----
  function conectar(url, alRecibir, alCambiarEstado) {
    var es = null, fallas = 0;
    function abrir() {
      es = new EventSource(url);
      es.onmessage = function (e) {
        fallas = 0;
        if (alCambiarEstado) alCambiarEstado(true, 0);
        try { alRecibir(JSON.parse(e.data)); } catch (err) {}
      };
      es.onerror = function () {
        fallas++;
        if (alCambiarEstado) alCambiarEstado(false, fallas);
      };
    }
    abrir();
    return { cerrar: function () { if (es) es.close(); } };
  }

  w.TT = { nombre: nombre, esc: esc, hhmm: hhmm, minutosDesde: minutosDesde, tocar: tocar, audio: audio,
    audioActivo: audioActivo, SONIDOS: SONIDOS, conectar: conectar };
})(window);
