/* TeToca · pantallas de Mesa de entradas, Caja y Configuración */
(function () {
  'use strict';
  const T = window.TT;
  const $ = (id) => document.getElementById(id);
  const esc = T.esc, nom = T.nombre;

  // ---------- identidad de esta PC ----------
  const params = new URLSearchParams(location.search);
  let PC = params.get('pc') || localStorage.getItem('tetoca-pc');
  if (!PC) { PC = 'PC-' + Math.random().toString(36).slice(2, 6).toUpperCase(); }
  try { localStorage.setItem('tetoca-pc', PC); } catch (e) {}
  const ES_PUESTO = params.get('puesto') === '1';

  let V = null;               // último estado recibido de la central
  let vista = null;           // 'mesa' | 'caja' | 'config' | 'rol'
  let montada = null;
  let filtro = 'todos', busqueda = '';
  let popTV = false, tvs = null, buscandoTV = false;
  let versionInicial = null;
  let tema = '', estilo = 'clasico';
  try { tema = localStorage.getItem('tetoca-tema') || ''; estilo = localStorage.getItem('tetoca-estilo') || 'clasico'; } catch (e) {}
  function guardarLocal(k, v) { try { localStorage.setItem(k, v); } catch (e) {} }

  // Temas del televisor: [id, nombre, descripción, fondo, barra, tarjeta, nombre, acento, panel]
  const TEMAS_TV = [
    ['noche', 'Noche', 'El de siempre. Oscuro, se lee bien en cualquier sala.', '#0A1120', '#0A1120', '#111B2E', '#fff', '#FFC857', '#0E1729'],
    ['celeste', 'Celeste y blanco', 'Barra celeste, fondo claro y un Sol de Mayo suave.', '#EEF4FA', '#74ACDF', '#fff', '#0F2A4A', '#1F5FA8', '#fff'],
    ['albiceleste', 'Albiceleste noche', 'Azul profundo con una franja celeste y blanca.', '#0B1B33', '#0B1B33', '#12284A', '#fff', '#F6B40E', '#0F2240'],
    ['sol', 'Sol de Mayo', 'Tonos cálidos, papel y dorado.', '#FBF6EA', '#FBF6EA', '#fff', '#2B2014', '#B07A00', '#fff'],
    ['claro', 'Claro neutro', 'Blanco y gris, sin colores fuertes.', '#F5F6F8', '#fff', '#fff', '#111827', '#2F6FDB', '#fff'],
    ['contraste', 'Alto contraste', 'Negro y amarillo, letras más grandes. Para salas con mucha luz o gente mayor.', '#000', '#000', '#000', '#fff', '#FFE600', '#000'],
    ['verde', 'Verde salud', 'Verde sereno y claro.', '#F1F7F4', '#1D6B5A', '#fff', '#12352D', '#1D6B5A', '#fff']
  ];

  // ---------- íconos ----------
  const I = {
    cast: '<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 8V6a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-6"/><path d="M2 12a9 9 0 0 1 8 8"/><path d="M2 16a5 5 0 0 1 4 4"/><path d="M2 20h.01"/></svg>',
    castOff: '<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 8V6a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-6"/><path d="M2 12a9 9 0 0 1 8 8"/><path d="M2 16a5 5 0 0 1 4 4"/><path d="M3 3l18 18"/></svg>',
    luna: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/></svg>',
    sol: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></svg>',
    engr: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/></svg>',
    subir: '<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 15V3"/><path d="m7 8 5-5 5 5"/><path d="M5 21h14"/></svg>',
    parlante: (c) => `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="${c}" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 5 6 9H2v6h4l5 4z"/><path d="M15.5 8.5a5 5 0 0 1 0 7"/><path d="M19 5a10 10 0 0 1 0 14"/></svg>`,
    flecha: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="m13 6 6 6-6 6"/></svg>',
    ok: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12 5 5 9-10"/></svg>',
    lupa: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>',
    tv: '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="13" rx="2"/><path d="M8 21h8"/><path d="M12 17v4"/></svg>',
    play: '<svg width="15" height="15" viewBox="0 0 24 24" fill="#fff"><path d="M7 4.5v15l13-7.5z"/></svg>',
    volver: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 12H5"/><path d="m11 6-6 6 6 6"/></svg>',
    act: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-3-6.7"/><path d="M21 4v5h-5"/></svg>'
  };

  // ---------- utilidades ----------
  function toast(msg, err) {
    const t = $('toast');
    t.textContent = msg; t.className = err ? 'err' : ''; t.style.display = 'block';
    clearTimeout(toast._t); toast._t = setTimeout(() => { t.style.display = 'none'; }, err ? 5000 : 2600);
  }
  async function api(path, body, metodo) {
    try {
      const r = await fetch(path, { method: metodo || 'POST', headers: { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) });
      const j = await r.json().catch(() => ({}));
      if (!r.ok) { toast(j.error || 'Algo salió mal.', true); return null; }
      return j;
    } catch (e) { toast('No hay conexión con la PC central.', true); return null; }
  }
  const accion = (a, id) => api('/api/accion', { accion: a, pc: PC, id: id || '' });
  function modal(html) { $('modal').innerHTML = html ? `<div class="modal" id="fondoModal"><div class="card">${html}</div></div>` : ''; }
  function confirmar(titulo, texto, si, fn, clase) {
    modal(`<h3>${titulo}</h3><p>${texto}</p><div class="grid2"><button class="btn" id="mNo">Cancelar</button><button class="btn ${clase || ''}" id="mSi" style="${clase ? '' : 'background:#2F6FDB;color:#fff;border:0'}">${si}</button></div>`);
    $('mNo').onclick = () => modal('');
    $('mSi').onclick = () => { modal(''); fn(); };
    $('fondoModal').onclick = (e) => { if (e.target.id === 'fondoModal') modal(''); };
  }
  const turnos = () => (V && V.estado.turnos) || [];
  const miConf = () => (V && V.config.pcs && V.config.pcs[PC]) || {};
  const miActual = () => turnos().find((t) => (t.estado === 'mesa' || t.estado === 'caja') && t.pc === PC);
  const hhmm = T.hhmm;
  function etiquetaPC(pc) {
    const c = V.config.pcs[pc];
    if (c && c.box) return c.box;
    return pc;
  }
  // ---------- dos registros / varios trámites por persona ----------
  const regs = () => (V && V.config.registros) || [];
  const dosReg = () => regs().length >= 2;
  const nomReg = (id) => regs()[+id - 1] || ('R' + id);
  const chipReg = (id) => `<span class="rg r${esc(id)}">${esc(nomReg(id))}</span>`;
  const chipsReg = (t) => (dosReg() ? (t.registros || []).map(chipReg).join('') : '');
  const cantTr = (t) => (t.tramites && t.tramites.length) || 1;
  const preCorta = (t) => (t.precarga || '') + (cantTr(t) > 1 ? ' +' + (cantTr(t) - 1) : '');
  const tramCorto = (t) => (cantTr(t) > 1 ? cantTr(t) + ' trámites' : (t.tramite || ''));
  const textoBusqueda = (t) => [t.nombre, t.precarga, t.dominio].concat((t.tramites || []).map((x) => x.precarga + ' ' + x.dominio)).join(' ');
  function tablaTramites(t) {
    if (cantTr(t) < 2) return `<div style="font-size:13px;color:var(--muted)">${esc(t.tramite || '')}</div>`;
    return `<div class="trs"><div class="trh"><span>${dosReg() ? 'REG.' : ''}</span><span>PRECARGA</span><span>TRÁMITE</span><span>DOMINIO</span></div>` +
      t.tramites.map((x) => `<div class="trf"><span>${dosReg() ? chipReg(x.registro) : ''}</span><span class="num" style="font-weight:700">${esc(x.precarga)}</span><span class="tt" title="${esc(x.tramite)}">${esc(x.tramite)}</span><span style="font-weight:700">${esc(x.dominio || '—')}</span></div>`).join('') + '</div>';
  }

  // Carga de los archivos del día: con dos registros, uno por registro.
  let cargaReg = '1';
  function cargarTurnos() {
    if (!dosReg()) { cargaReg = '1'; $('archivo').click(); return; }
    modalCarga();
  }
  function modalCarga() {
    const fu = (V.estado.cargadoDia === V.hoy && V.estado.fuentes) || {};
    const juntos = turnos().filter((t) => cantTr(t) > 1).length;
    modal(`<h3>Cargar turnos del día</h3><p>Un archivo por registro. Se pueden cargar en cualquier orden; si se vuelve a cargar uno, se actualiza solo ese.</p>` +
      ['1', '2'].map((id) => {
        const f = fu[id];
        return `<div class="slot">${chipReg(id)}<div style="flex:1;min-width:0">${f ? `<b>${esc(f.archivo)}</b><small class="ok">✓ ${f.cantidad} turnos cargados a las ${hhmm(f.cargado)}</small>` : '<b>Sin cargar</b><small>Todavía no se cargó el archivo de este registro.</small>'}</div>
          <button class="btn" data-cargar="${id}" style="${f ? '' : 'background:#2F6FDB;color:#fff;border:0;'}padding:0 14px">${f ? 'Cambiar' : 'Elegir archivo'}</button></div>`;
      }).join('') +
      (juntos ? `<div class="nota" style="margin:0;background:var(--hl);color:var(--ink);border-radius:12px;padding:12px 14px"><strong>${juntos} ${juntos === 1 ? 'persona tiene' : 'personas tienen'} más de un turno en el mismo horario</strong> (casi siempre gestores): quedaron en una sola fila, con todos sus trámites.</div>` : '') +
      `<button class="btn" id="mSi" style="background:#2F6FDB;color:#fff;border:0">Listo</button>`);
    document.querySelectorAll('[data-cargar]').forEach((b) => { b.onclick = () => { cargaReg = b.dataset.cargar; $('archivo').click(); }; });
    $('mSi').onclick = () => modal('');
    $('fondoModal').onclick = (e) => { if (e.target.id === 'fondoModal') modal(''); };
  }

  const ESTADO = { pendiente: 'Pendiente', mesa: 'En mesa', espera_caja: 'Espera caja', caja: 'En caja', terminado: 'Terminado', ausente: 'Ausente', cancelado: 'Cancelado' };
  function pill(t) {
    let txt = ESTADO[t.estado] || t.estado;
    if ((t.estado === 'mesa' || t.estado === 'caja') && t.pc && t.pc !== PC) txt += ' · ' + etiquetaPC(t.pc);
    if (t.estado === 'ausente') txt += t.ausenteEn === 'caja' ? ' (caja)' : '';
    return `<span class="pill p-${t.estado}">${esc(txt)}</span>`;
  }

  let esperandoPrueba = 0;
  function resultadoSonidoTV() {
    const r = V.tvSonido;
    if (esperandoPrueba && (!r || !r.prueba)) {
      if (Date.now() - esperandoPrueba > 8000) return '<span style="color:var(--danger);font-weight:700">El TV no respondió. ¿Está mostrando la pantalla de sala?</span>';
      setTimeout(() => { if (vista === 'config' && !enEdicion()) vistaConfig(); }, 1500);
      return '<span style="color:var(--muted)">Esperando al TV…</span>';
    }
    if (!r || !r.prueba) return '<span style="color:var(--muted)">Hace sonar el llamado en el televisor y avisa si salió bien.</span>';
    return r.ok ? `<span style="color:var(--okFg);font-weight:700">✓ El TV reprodujo el sonido.</span> <span style="color:var(--muted)">Si no se escuchó, es el volumen del TV o del Chromecast.</span>`
      : `<span style="color:var(--danger);font-weight:700">✗ El TV no pudo reproducir el sonido.</span> <span style="color:var(--muted)">${esc(r.detalle)}</span>`;
  }

  // La instalación la maneja el programa de cada PC: en un puesto, su ayudante local
  // (127.0.0.1:8767); en la central, la propia central. Desde otra PC no se muestra.
  function urlInstalacion() {
    const volver = encodeURIComponent(location.href);
    if (/[?&]puesto=1/.test(location.search)) return 'http://127.0.0.1:8767/instalacion?volver=' + volver;
    if (location.hostname === '127.0.0.1' || location.hostname === 'localhost') return '/instalacion?volver=' + volver;
    return '';
  }

  // ---------- tema ----------
  function aplicarTema() {
    const t = tema || (V && V.config.tema) || 'claro';
    document.body.classList.toggle('oscuro', t === 'oscuro');
    document.body.classList.toggle('t-argentina', estilo === 'argentina');
    return t;
  }

  // ---------- barra superior ----------
  function barra() {
    const c = V.config, rol = vista === 'config' ? 'config' : vista;
    const tv = V.tv || {}, conectado = tv.estado === 'conectado';
    let tvTxt = 'Conectar TV', tvCls = 'off', tvIco = I.castOff;
    if (conectado) { tvTxt = '<span class="dot"></span>TV conectado'; tvCls = ''; tvIco = I.cast; }
    else if (tv.estado === 'conectando') { tvTxt = 'Conectando TV…'; tvCls = ''; tvIco = I.cast; }
    else if (tv.estado === 'ocupado') { tvTxt = 'TV ocupado'; }
    else if (tv.estado === 'error') { tvTxt = 'TV sin conexión'; }
    const ahora = new Date();
    const temaAct = aplicarTema();
    $('barra').innerHTML = `
      <div class="marca"><img src="/logo.png?v=${c.tieneLogo ? 1 : 0}" alt="">${c.organismo ? `<span class="org">${esc(c.organismo)}</span>` : ''}</div>
      <span class="sep"></span><span class="app">TeToca</span>
      ${rol === 'mesa' ? '<span class="rol mesa">Mesa de entradas</span>' : rol === 'caja' ? '<span class="rol caja">Caja</span>' : rol === 'config' ? '<span class="rol config">Configuración</span>' : ''}
      ${miConf().mostrarBox && miConf().box && rol !== 'config' ? `<span class="rol config">${esc(miConf().box)}</span>` : ''}
      <span class="esp"></span>
      <span class="con" title="Conectado con la PC central"><span class="dot"></span><span class="lbl">Conectado · ${Math.max(1, V.equipos)} ${V.equipos === 1 ? 'equipo' : 'equipos'}</span></span>
      <span class="sep"></span>
      <span class="hora"><span class="lbl">${ahora.toLocaleDateString('es-AR', { weekday: 'short', day: '2-digit', month: '2-digit' })} · </span><b>${ahora.toLocaleTimeString('es-AR', { hour: '2-digit', minute: '2-digit', hour12: false })}</b></span>
      <button class="bb tv ${tvCls} ${popTV ? 'act' : ''}" id="bTV" title="Transmitir al televisor">${tvIco}${tvTxt}</button>
      <button class="bb" id="bTema" aria-label="Cambiar tema" title="Cambiar tema">${temaAct === 'oscuro' ? I.sol + '<span class="lbl">Tema claro</span>' : I.luna + '<span class="lbl">Tema oscuro</span>'}</button>
      ${rol === 'config' ? `<button class="bb" id="bVolver">${I.volver}Volver</button>` : `<button class="bb" id="bCfg" title="Configuración">${I.engr}<span class="lbl">Configuración</span></button>`}
      ${rol === 'mesa' ? `<button class="bb" id="bCargar" title="Cargar turnos del día (XLS)">${I.subir}<span class="lbl">Cargar turnos del día (XLS)</span></button>` : ''}`;
    $('bTV').onclick = () => { popTV = !popTV; if (popTV && !tvs) buscarTVs(); render(); };
    $('bTema').onclick = () => { tema = temaAct === 'oscuro' ? 'claro' : 'oscuro'; guardarLocal('tetoca-tema', tema); render(); };
    if ($('bCfg')) $('bCfg').onclick = () => irA('config');
    if ($('bVolver')) $('bVolver').onclick = () => irA(miConf().rol || 'rol');
    if ($('bCargar')) $('bCargar').onclick = () => cargarTurnos();
  }

  // ---------- avisos (actualización, fecha) ----------
  function avisos() {
    const u = V.update || {}, e = V.estado;
    let h = '';
    const descartada = localStorage.getItem('tetoca-upd-mas-tarde');
    if (u.estado === 'descargando' || u.estado === 'reiniciando') {
      h += `<div class="banner">${I.act}<span><strong>${esc(u.mensaje)}</strong> Las pantallas se recargan solas en unos segundos.</span></div>`;
    } else if (u.estado === 'error') {
      h += `<div class="banner err">${I.act}<span>${esc(u.mensaje)}</span></div>`;
    } else if (u.disponible && descartada !== u.version) {
      h += `<div class="banner">${I.act}<span><strong>Hay una actualización disponible</strong> · versión ${esc(u.version)}${u.notas ? ' — ' + esc(u.notas.split('\n')[0]) : ''}</span><span class="esp"></span>
        <button class="b1" id="uTarde">Más tarde</button><button class="b2" id="uYa">Actualizar ahora</button></div>`;
    }
    if (turnos().length && e.fecha && e.fecha !== V.hoy) {
      h += `<div class="banner warn"><span>Estos turnos son del <strong>${esc(e.fecha)}</strong> y hoy es ${esc(V.hoy)}. Cargá el archivo de hoy.</span></div>`;
    }
    $('avisos').innerHTML = h;
    if ($('uTarde')) $('uTarde').onclick = () => { localStorage.setItem('tetoca-upd-mas-tarde', u.version); avisos(); };
    if ($('uYa')) $('uYa').onclick = () => confirmar('Actualizar TeToca', 'Se baja la versión nueva y el sistema se reinicia en unos segundos. Todas las pantallas se recargan solas. No se pierde nada de la lista del día.', 'Actualizar ahora', () => api('/api/update/aplicar', {}));
  }

  // ---------- panel del TV ----------
  async function buscarTVs() {
    buscandoTV = true; render();
    const r = await api('/api/tv/buscar', undefined, 'GET');
    tvs = r || []; buscandoTV = false; render();
  }
  function panelTV() {
    if (!popTV) { $('pop').innerHTML = ''; return; }
    const cfg = V.config.tv || {}, st = V.tv || {};
    let filas = '';
    if (cfg.host) {
      const on = st.estado === 'conectado';
      filas += `<div class="tvrow ${on ? 'on' : ''}"><span style="display:flex;color:${on ? 'var(--okFg)' : 'var(--muted)'}">${I.tv}</span>
        <div style="flex:1;min-width:0"><div class="n">${esc(cfg.nombre || cfg.host)}</div><div class="d">${on ? '● ' : ''}${esc(st.mensaje || '')}</div></div>
        ${cfg.auto ? `<button class="btn" style="height:36px;padding:0 12px" id="tvDesc">Desconectar</button>` : `<button class="bazul" id="tvRe">Transmitir</button>`}</div>`;
    }
    (tvs || []).filter((t) => t.id !== cfg.id).forEach((t, i) => {
      filas += `<div class="tvrow"><span style="display:flex;color:var(--muted)">${I.tv}</span>
        <div style="flex:1;min-width:0"><div class="n">${esc(t.nombre)}</div><div class="d">${esc(t.modelo || 'Disponible')}</div></div>
        <button class="bazul" data-tv="${i}">Transmitir</button></div>`;
    });
    if (!filas) filas = `<div class="vacio">${buscandoTV ? 'Buscando televisores en la red…' : 'No encontré televisores. Revisá que el TV esté prendido y en la misma red.'}</div>`;
    $('pop').innerHTML = `<div class="pop">
      <div class="fila"><span style="display:flex;gap:10px;align-items:center;font-size:17px;font-weight:800">${I.cast} Transmitir al televisor</span><button class="link" id="tvCerrar" style="font-size:22px;text-decoration:none">×</button></div>
      <div style="font-size:13px;color:var(--muted)">Televisores encontrados en la red${buscandoTV ? ' · buscando…' : ''}</div>
      ${filas}
      <div class="fila" style="border-top:1px solid var(--line2);padding-top:12px">
        <button class="link" id="tvBuscar" style="color:#2F6FDB;font-weight:700;text-decoration:none">Buscar televisores de nuevo</button>
        <span style="font-size:12px;color:var(--muted)">Si se corta, se reconecta solo</span></div>
      <div class="nota" style="margin:0">¿El TV no muestra nada? <a href="/tv" target="_blank" style="color:inherit">Abrí la pantalla de sala en el navegador</a> y transmitila desde Chrome (⋮ › Transmitir).</div>
    </div>`;
    $('tvCerrar').onclick = () => { popTV = false; render(); };
    $('tvBuscar').onclick = buscarTVs;
    if ($('tvDesc')) $('tvDesc').onclick = () => api('/api/tv/desconectar', {});
    if ($('tvRe')) $('tvRe').onclick = () => api('/api/tv/reconectar', {});
    document.querySelectorAll('[data-tv]').forEach((b) => {
      b.onclick = () => { const t = tvs[+b.dataset.tv]; api('/api/tv/conectar', t).then((r) => r && toast('Conectando con ' + t.nombre + '…')); };
    });
  }

  // ---------- elegir rol ----------
  function vistaRol() {
    $('vista').innerHTML = `<div class="elegir">
      <h1>¿Qué es esta computadora?</h1>
      <p style="margin:0;color:var(--muted)">Se elige una sola vez. Después se puede cambiar desde Configuración.</p>
      <div class="opciones">
        <button class="opcion" id="rMesa">${I.parlante('#2F6FDB')}<b>Mesa de entradas</b><span>Llama a la gente por turno y la pasa a caja si tiene que pagar.</span></button>
        <button class="opcion" id="rCaja">${I.parlante('#B8860B')}<b>Caja</b><span>Llama a los que vienen de mesa, por orden de llegada, y cobra.</span></button>
      </div>
      <p class="nota">Nombre de esta PC: <strong>${esc(PC)}</strong></p></div>`;
    $('rMesa').onclick = () => api('/api/pc', { pc: PC, rol: 'mesa' }).then(() => irA('mesa'));
    $('rCaja').onclick = () => api('/api/pc', { pc: PC, rol: 'caja' }).then(() => irA('caja'));
  }

  // ---------- tarjeta "atendiendo ahora" ----------
  function tarjetaActual(t, esCaja) {
    if (!t) {
      return `<div class="tit">${esCaja ? 'COBRANDO AHORA' : 'ATENDIENDO AHORA'}</div>
        <div class="vacio" style="padding:26px 0 8px">Nadie llamado. Tocá <strong>Llamar siguiente</strong> o elegí a alguien de la lista.</div>`;
    }
    const llam = esCaja ? t.llamadoCaja : t.llamadoMesa;
    const esp = esCaja ? T.minutosDesde(t.pasoCaja, llam) : null;
    return `<div class="fila"><span class="tit">${esCaja ? 'COBRANDO AHORA' : 'ATENDIENDO AHORA'}</span>
        <span class="pill ${esCaja ? 'p-caja' : 'p-mesa'}">Llamado ${hhmm(llam)}</span></div>
      <div><div class="grande num">${esc(t.precarga || '—')}</div><div class="nomg">${esc(nom(t.nombre))}</div>
        ${cantTr(t) > 1 || dosReg() ? `<div style="display:flex;align-items:center;gap:6px;margin-top:8px;flex-wrap:wrap">${chipsReg(t)}${cantTr(t) > 1 ? `<span style="font-size:13px;font-weight:700;color:var(--ink2)">${cantTr(t)} trámites</span>` : ''}</div>` : ''}</div>
      <div class="datos">
        ${esCaja ? `<div><small>DOMINIO</small><b>${esc(cantTr(t) > 1 ? 'Ver abajo' : (t.dominio || '—'))}</b></div><div><small>DE MESA</small><b>${hhmm(t.pasoCaja) || '—'}</b></div><div><small>ESPERÓ</small><b>${esp == null ? '—' : esp + ' min'}</b></div>`
                : `<div><small>TURNO</small><b>${esc(t.hora || '—')}</b></div><div><small>DOMINIO</small><b>${esc(cantTr(t) > 1 ? 'Ver abajo' : (t.dominio || '—'))}</b></div><div><small>ESTADO</small><b>En mesa</b></div>`}
      </div>
      ${tablaTramites(t)}
      ${esCaja ? `<button class="btn prim verde" id="aTerm">${I.ok}Cobro terminado</button>
        <div class="grid2"><button class="btn" id="aRe">Volver a llamar</button><button class="btn rojo" id="aAus">Ausente</button></div>`
              : `<button class="btn prim amarillo" id="aCaja">Pasar a caja ${I.flecha}</button>
        <div class="grid3"><button class="btn" id="aTerm">Terminado</button><button class="btn" id="aRe">Rellamar</button><button class="btn rojo" id="aAus">Ausente</button></div>`}
      <div class="fila" style="justify-content:flex-start;gap:18px"><button class="link" id="aError">Lo llamé por error</button>
      ${!esCaja && cantTr(t) > 1 ? '<button class="link" id="aSeparar">No es la misma persona: separar</button>' : ''}</div>`;
  }
  function enlazarActual() {
    if ($('aCaja')) $('aCaja').onclick = () => accion('pasar_caja').then((r) => r && toast('Pasó a la fila de caja.'));
    if ($('aTerm')) $('aTerm').onclick = () => accion('terminar');
    if ($('aRe')) $('aRe').onclick = () => accion('rellamar');
    if ($('aAus')) $('aAus').onclick = () => accion('ausente');
    if ($('aSeparar')) $('aSeparar').onclick = () => { const a = miActual(); if (a) confirmar('Separar turnos', 'Se van a mostrar como personas distintas. Seguís atendiendo el primer trámite; los otros vuelven a la lista.', 'Separar', () => api('/api/separar', { id: a.id }).then((r) => r && toast('Listo, quedaron separados.'))); };
    if ($('aError')) $('aError').onclick = () => accion('deshacer_llamado').then((r) => r && toast('Listo, volvió a su lugar en la lista.'));
  }

  // ---------- MESA ----------
  function montarMesa() {
    $('vista').innerHTML = `<main>
      <div class="col">
        <div class="card pad" id="mActual" style="display:flex;flex-direction:column;gap:14px"></div>
        <div id="mSig"></div>
        <div class="card pad scroll" id="mAus" style="padding:18px 20px;flex:1"></div>
      </div>
      <div class="card lista">
        <div class="tool">
          <label class="buscar">${I.lupa}<input id="q" placeholder="Buscar nombre, precarga o dominio" aria-label="Buscar"></label>
          <div class="seg" id="filtros"></div>
          <button class="chico" id="bManual">+ Agregar</button>
        </div>
        <div class="thead"><span>HORA</span><span>PRECARGA</span><span>SOLICITANTE</span><span>TRÁMITE</span><span>DOMINIO</span><span>ESTADO</span></div>
        <div class="scroll" id="rows" style="flex:1"></div>
      </div></main>`;
    $('q').value = busqueda;
    $('q').oninput = () => { busqueda = $('q').value; filasMesa(); };
    $('bManual').onclick = agregarManual;
  }
  function actualizarMesa() {
    const yo = miActual();
    $('mActual').innerHTML = tarjetaActual(yo && yo.estado === 'mesa' ? yo : null, false);
    enlazarActual();
    const sig = turnos().find((t) => t.estado === 'pendiente');
    $('mSig').innerHTML = `<button class="sig azul" id="bSig" ${yo || !sig ? 'disabled' : ''}>
      <span class="ic">${I.parlante('#fff')}</span>
      <span style="flex:1"><span class="t1">Llamar siguiente</span><span class="t2">${sig ? esc((sig.precarga ? preCorta(sig) + ' · ' : '') + nom(sig.nombre) + ' · ' + sig.hora) : 'No quedan turnos pendientes'}</span></span>
      <span class="kbd">ESPACIO</span></button>`;
    $('bSig').onclick = siguiente;
    const aus = turnos().filter((t) => t.estado === 'ausente' && t.ausenteEn !== 'caja').sort((a, b) => new Date(b.ausenteA) - new Date(a.ausenteA));
    $('mAus').innerHTML = `<div class="fila"><span class="tit">AUSENTES</span><span class="pill p-ausente">${aus.length}</span></div>` +
      (aus.length ? aus.map((t) => `<div class="mini"><div style="flex:1;min-width:0"><div class="n">${esc(nom(t.nombre))}</div><div class="d">${esc(preCorta(t))} · turno ${esc(t.hora)} · llamado ${hhmm(t.ausenteA)}</div></div><button class="chico" data-llamar="${esc(t.id)}" ${yo ? 'disabled' : ''}>Llamar</button></div>`).join('')
                  : '<div class="vacio">Nadie ausente.</div>');
    $('mAus').querySelectorAll('[data-llamar]').forEach((b) => { b.onclick = () => accion('mesa_llamar', b.dataset.llamar); });
    filasMesa();
  }
  const FILTROS = [['todos', 'Todos', () => true], ['pend', 'Pendientes', (t) => t.estado === 'pendiente'], ['aten', 'En mesa', (t) => t.estado === 'mesa'],
    ['caja', 'En caja', (t) => t.estado === 'espera_caja' || t.estado === 'caja'], ['term', 'Terminados', (t) => t.estado === 'terminado'], ['aus', 'Ausentes', (t) => t.estado === 'ausente']];
  function filtros() {
    if (!dosReg()) return FILTROS;
    return FILTROS.slice(0, 1).concat([['r1', nomReg(1), (t) => (t.registros || []).includes('1')], ['r2', nomReg(2), (t) => (t.registros || []).includes('2')]], FILTROS.slice(1));
  }
  function filasMesa() {
    const ts = turnos();
    const FILTROS = filtros();
    if (!FILTROS.find((x) => x[0] === filtro)) filtro = 'todos';
    $('filtros').innerHTML = FILTROS.map(([k, n, f]) => `<button data-f="${k}" class="${filtro === k ? 'on' : ''}">${n} ${ts.filter(f).length}</button>`).join('');
    $('filtros').querySelectorAll('button').forEach((b) => { b.onclick = () => { filtro = b.dataset.f; filasMesa(); }; });
    if (!ts.length) {
      $('rows').innerHTML = `<div class="vac-lista"><b>Todavía no se cargaron los turnos de hoy</b><span>Descargá el archivo del sistema de turnos y cargalo acá.</span>
        <button class="btn prim" style="background:#2F6FDB;color:#fff;padding:0 22px" id="bCargar2">${I.subir}Cargar turnos del día (XLS)</button></div>`;
      $('bCargar2').onclick = () => cargarTurnos();
      return;
    }
    const f = FILTROS.find((x) => x[0] === filtro)[2];
    const q = busqueda.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '');
    const lista = ts.filter(f).filter((t) => !q || textoBusqueda(t).toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '').includes(q));
    const yo = miActual();
    $('rows').innerHTML = lista.map((t) => `<div class="tr ${yo && yo.id === t.id ? 'act' : ''} ${t.estado === 'terminado' || t.estado === 'cancelado' ? 'fin' : ''}" data-id="${esc(t.id)}">
        <span class="num" style="font-weight:700">${esc(t.hora)}</span><span class="num dm">${esc(preCorta(t))}</span>
        <span class="n" style="${t.estado === 'cancelado' ? 'text-decoration:line-through' : ''}">${chipsReg(t)}${esc(nom(t.nombre))}</span>
        <span class="tm" title="${esc(cantTr(t) > 1 ? t.tramites.map((x) => x.tramite).join(' · ') : t.tramite)}" style="${cantTr(t) > 1 ? 'color:var(--ink);font-weight:700' : ''}">${esc(tramCorto(t))}</span><span class="dm">${esc(cantTr(t) > 1 ? (t.tramites.map((x) => x.dominio).filter(Boolean)[0] || '—') + (t.tramites.filter((x) => x.dominio).length > 1 ? ' +' + (t.tramites.filter((x) => x.dominio).length - 1) : '') : (t.dominio || '—'))}</span><span>${pill(t)}</span></div>`).join('') ||
      '<div class="vacio" style="padding:20px">No hay turnos con ese filtro.</div>';
    $('rows').querySelectorAll('.tr').forEach((r) => { r.onclick = () => clickTurno(r.dataset.id); });
  }
  function clickTurno(id) {
    const t = turnos().find((x) => x.id === id);
    if (!t) return;
    const yo = miActual();
    if (yo && yo.id === t.id) { accion('rellamar'); return; }
    if (t.estado === 'cancelado') { toast('Ese turno figura como cancelado.'); return; }
    if (t.estado === 'mesa' || t.estado === 'caja') { toast(nom(t.nombre) + ' ya lo está atendiendo ' + etiquetaPC(t.pc) + '.'); return; }
    if (t.estado === 'espera_caja') { toast(nom(t.nombre) + ' está en la fila de caja.'); return; }
    if (t.estado === 'ausente' && t.ausenteEn === 'caja') { toast(nom(t.nombre) + ' quedó ausente en caja: lo llama la caja.'); return; }
    if (yo) { toast('Primero indicá qué pasó con ' + nom(yo.nombre) + '.', true); return; }
    const extra = t.estado === 'terminado' ? ' Ya figura como terminado.' : '';
    confirmar('¿Llamar a esta persona?', `<strong style="color:var(--ink)">${esc(nom(t.nombre))}</strong> · precarga ${esc(preCorta(t))}${cantTr(t) > 1 ? ' (' + cantTr(t) + ' trámites)' : ''} · turno ${esc(t.hora)}.${extra}`, 'Llamar', () => accion('mesa_llamar', t.id));
  }
  function agregarManual() {
    modal(`<h3>Agregar turno a mano</h3><p>Para alguien que no está en el archivo del día.</p>
      <div class="campo" style="margin:0"><label for="mNom" style="width:90px">Nombre</label><input type="text" id="mNom" placeholder="APELLIDO, NOMBRE"></div>
      <div class="campo" style="margin:0"><label for="mPre" style="width:90px">Precarga</label><input type="text" id="mPre" placeholder="Opcional"></div>
      <div class="grid2"><button class="btn" id="mNo">Cancelar</button><button class="btn" id="mSi" style="background:#2F6FDB;color:#fff;border:0">Agregar</button></div>`);
    $('mNom').focus();
    $('mNo').onclick = () => modal('');
    $('mSi').onclick = () => { const n = $('mNom').value.trim(); if (!n) return; api('/api/manual', { nombre: n, precarga: $('mPre').value }).then((r) => { if (r) { modal(''); toast('Agregado a la lista.'); } }); };
  }

  // ---------- CAJA ----------
  function montarCaja() {
    $('vista').innerHTML = `<main>
      <div class="col">
        <div class="card pad" id="cActual" style="display:flex;flex-direction:column;gap:14px"></div>
        <div id="cSig"></div>
        <div class="grid2" id="cStats" style="gap:16px"></div>
      </div>
      <div class="col">
        <div class="card lista" style="flex:1">
          <div class="fila" style="padding:18px 22px;border-bottom:1px solid var(--line2)">
            <span style="display:flex;align-items:center;gap:12px"><span style="font-size:18px;font-weight:800">Fila de caja</span><span class="pill p-caja" id="cCant"></span></span>
            <span style="font-size:13px;color:var(--muted)">Ordenada por llegada desde mesa</span></div>
          <div class="scroll" id="fila" style="flex:1"></div>
        </div>
        <div class="grid2" style="gap:16px">
          <div class="card pad scroll" id="cAus" style="padding:18px 22px;max-height:230px"></div>
          <div class="card pad scroll" id="cCob" style="padding:18px 22px;max-height:230px"></div>
        </div>
      </div></main>`;
  }
  function actualizarCaja() {
    const yo = miActual();
    $('cActual').innerHTML = tarjetaActual(yo && yo.estado === 'caja' ? yo : null, true);
    enlazarActual();
    const fila = turnos().filter((t) => t.estado === 'espera_caja').sort((a, b) => new Date(a.pasoCaja) - new Date(b.pasoCaja));
    const sig = fila[0];
    $('cSig').innerHTML = `<button class="sig amarillo" id="bSig" ${yo || !sig ? 'disabled' : ''}>
      <span class="ic">${I.parlante('#16181D')}</span>
      <span style="flex:1"><span class="t1">Llamar siguiente</span><span class="t2">${sig ? esc((sig.precarga ? preCorta(sig) + ' · ' : '') + nom(sig.nombre) + ' · llegó ' + hhmm(sig.pasoCaja)) : 'No hay nadie esperando'}</span></span>
      <span class="kbd">ESPACIO</span></button>`;
    $('bSig').onclick = siguiente;
    $('cCant').textContent = fila.length + ' esperando';
    $('fila').innerHTML = fila.map((t, i) => `<div class="fc"><span class="pos num ${i === 0 ? 'uno' : ''}">${i + 1}</span>
        <div style="flex:1;min-width:0"><div style="font-size:16px;font-weight:800">${chipsReg(t)}${esc(nom(t.nombre))}</div>
        <div style="font-size:13px;color:var(--muted);margin-top:3px"><span class="num" style="color:var(--ink2);font-weight:600">${esc(preCorta(t))}</span> · ${esc(tramCorto(t))}${t.dominio && cantTr(t) < 2 ? ' · ' + esc(t.dominio) : ''}</div></div>
        <div style="text-align:right;flex-shrink:0"><div style="font-size:14px;font-weight:700">Llegó ${hhmm(t.pasoCaja)}</div><div style="font-size:12px;color:var(--muted);margin-top:2px">espera ${T.minutosDesde(t.pasoCaja)} min</div></div>
        <button class="chico" data-llamar="${esc(t.id)}" ${yo ? 'disabled' : ''}>Llamar</button></div>`).join('') ||
      '<div class="vac-lista"><b>No hay nadie en la fila</b><span>Cuando mesa de entradas pase a alguien a caja, aparece acá.</span></div>';
    const aus = turnos().filter((t) => t.estado === 'ausente' && t.ausenteEn === 'caja').sort((a, b) => new Date(b.ausenteA) - new Date(a.ausenteA));
    $('cAus').innerHTML = `<div class="fila"><span class="tit">AUSENTES EN CAJA</span><span class="pill p-ausente">${aus.length}</span></div>` +
      (aus.map((t) => `<div class="mini"><div style="flex:1;min-width:0"><div class="n">${esc(nom(t.nombre))}</div><div class="d">${esc(preCorta(t))} · llamado ${hhmm(t.ausenteA)}</div></div><button class="chico" data-llamar="${esc(t.id)}" ${yo ? 'disabled' : ''}>Llamar</button></div>`).join('') || '<div class="vacio">Nadie ausente.</div>');
    const cob = turnos().filter((t) => t.estado === 'terminado' && t.llamadoCaja && t.llamadoCaja.indexOf('0001-') !== 0).sort((a, b) => new Date(b.termino) - new Date(a.termino));
    $('cCob').innerHTML = `<div class="tit" style="margin-bottom:6px">ÚLTIMOS COBRADOS</div>` +
      (cob.slice(0, 6).map((t) => `<div class="fila" style="font-size:14px;padding:4px 0"><span style="font-weight:700;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(nom(t.nombre))}</span><span style="color:var(--muted)">${hhmm(t.termino)}</span></div>`).join('') || '<div class="vacio">Todavía nadie.</div>');
    const esperas = turnos().filter((t) => t.llamadoCaja && t.pasoCaja && t.llamadoCaja.indexOf('0001-') !== 0 && t.pasoCaja.indexOf('0001-') !== 0)
      .map((t) => (new Date(t.llamadoCaja) - new Date(t.pasoCaja)) / 60000);
    const prom = esperas.length ? Math.round(esperas.reduce((a, b) => a + b, 0) / esperas.length) : null;
    $('cStats').innerHTML = `<div class="card pad" style="padding:18px 20px"><div class="tit">COBRADOS HOY</div><div class="stat num">${cob.length}</div></div>
      <div class="card pad" style="padding:18px 20px"><div class="tit">ESPERA PROMEDIO</div><div class="stat num">${prom == null ? '—' : prom + ' <span style="font-size:18px;color:var(--muted)">min</span>'}</div></div>`;
    document.querySelectorAll('#fila [data-llamar], #cAus [data-llamar]').forEach((b) => { b.onclick = () => accion('caja_llamar', b.dataset.llamar); });
  }

  function siguiente() {
    if (vista === 'mesa') accion('mesa_siguiente');
    else if (vista === 'caja') accion('caja_siguiente');
  }

  // ---------- CONFIGURACIÓN ----------
  let sonando = null;
  function vistaConfig() {
    const c = V.config, yo = miConf(), u = V.update || {}, temaAct = aplicarTema();
    const son = T.SONIDOS.map(([id, n, d], i) => `<div class="son ${c.sonido === id ? 'on' : ''}">
        <div class="fila"><span class="nn num" style="${sonando === id ? 'background:#FFC857;color:#16181D' : ''}">${i + 1}</span>${c.sonido === id ? '<span class="pill p-terminado">Elegido</span>' : ''}</div>
        <div><div style="font-size:18px;font-weight:800">${n}</div><div style="font-size:13px;color:var(--muted);margin-top:4px;line-height:1.45">${d}</div></div>
        <div style="display:flex;gap:8px;margin-top:auto"><button class="play" data-play="${id}">${I.play}${sonando === id ? 'Sonando…' : 'Escuchar'}</button>
        <button class="btn" style="width:104px;${c.sonido === id ? 'background:var(--okBg);color:var(--okFg);border-color:transparent' : ''}" data-elegir="${id}">${c.sonido === id ? 'Elegido' : 'Elegir'}</button></div></div>`).join('');
    const destino = (yo.rol === 'caja' ? 'CAJA' : 'MESA DE ENTRADAS') + (yo.mostrarBox && yo.box ? ' · ' + yo.box.toUpperCase() : '');
    $('vista').innerHTML = `<div class="cfg">
      <div><h1>Configuración</h1><p class="s">Los cambios se guardan solos y llegan a todas las pantallas.</p></div>
      <div><h2>Sonido de llamado</h2><p class="s">Suena en el televisor cada vez que se llama a alguien. Tocá "Escuchar" para probar.</p></div>
      <div class="sons">${son}</div>
      <div class="cfgrid">
        <div class="card pad"><h2>Esta PC</h2><p class="s">Nombre: <strong style="color:var(--ink)">${esc(PC)}</strong></p>
          <div class="campo"><label>Esta PC es</label><div class="segc"><button data-rol="mesa" class="${yo.rol === 'mesa' ? 'on' : ''}">Mesa de entradas</button><button data-rol="caja" class="${yo.rol === 'caja' ? 'on' : ''}">Caja</button></div></div>
          <div class="campo"><label>Estilo</label><div class="segc"><button data-estilo="clasico" class="${estilo !== 'argentina' ? 'on' : ''}">Clásico</button><button data-estilo="argentina" class="${estilo === 'argentina' ? 'on' : ''}">Celeste y blanco</button></div></div>
          <div class="campo"><label>Tema</label><div class="segc"><button data-tema="claro" class="${temaAct === 'claro' ? 'on' : ''}">${I.sol} Claro</button><button data-tema="oscuro" class="${temaAct === 'oscuro' ? 'on' : ''}">${I.luna} Oscuro</button></div></div>
          <div class="campo"><label for="cBox">Nombre del box</label><input type="text" id="cBox" value="${esc(yo.box || '')}" placeholder="Ej: Box 2">
            <button class="sw ${yo.mostrarBox ? 'on' : ''}" id="cMostrar" aria-label="Mostrar box en el TV"><i></i></button></div>
          <p class="nota">Con el interruptor prendido, el TV dice: <strong style="color:var(--ink)">Diríjase a ${esc(destino)}</strong></p></div>
        <div class="card pad"><h2>Pantalla de la sala</h2><p class="s">Lo que se ve arriba a la izquierda del TV y de estas pantallas.</p>
          <div class="campo"><label>Logo</label><img src="/logo.png?t=${Date.now()}" alt="" style="width:40px;height:40px;border-radius:50%;object-fit:cover"><button class="chico" id="cLogo">Cambiar logo</button>${c.tieneLogo ? '<button class="link" id="cSinLogo">Quitar</button>' : ''}</div>
          <div class="campo"><label for="cOrg">Texto al lado del logo</label><input type="text" id="cOrg" value="${esc(c.organismo || '')}" placeholder="Ej: DNRPA"></div>
          <div class="campo"><label for="cOfi">Nombre de la oficina</label><input type="text" id="cOfi" value="${esc(c.oficina || '')}" placeholder="Ej: Registro Automotor"></div>
          <div class="campo"><label for="cSec">Seccional</label><input type="text" id="cSec" value="${esc(c.seccional || '')}" placeholder="Ej: Azul 1 y 2"></div>
          <div class="campo"><label for="cUlt">Últimos llamados</label><input type="number" id="cUlt" min="1" max="8" value="${c.ultimos}" style="max-width:90px">
            <label for="cAusN" style="width:auto">Ausentes</label><input type="number" id="cAusN" min="0" max="6" value="${c.ausentes}" style="max-width:90px"></div></div>
        <div class="card pad"><h2>Turnos del día</h2><p class="s">${turnos().length ? `Cargados: ${turnos().length} turnos del ${esc(V.estado.fecha)}${V.estado.archivo ? ' (' + esc(V.estado.archivo) + ')' : ''}.` : 'Todavía no se cargaron.'}</p>
          <div style="display:flex;gap:10px;margin-top:14px"><button class="btn" style="background:#2F6FDB;color:#fff;border:0;padding:0 18px" id="cCargar">${I.subir} Cargar turnos del día (XLS)</button><button class="btn" style="padding:0 18px" id="cManual">+ Agregar a mano</button></div>
          <div class="campo"><label>Registros</label><div class="segc"><button data-nreg="1" class="${dosReg() ? '' : 'on'}">Uno</button><button data-nreg="2" class="${dosReg() ? 'on' : ''}">Dos, en la misma mesa</button></div></div>
          ${dosReg() ? `<div class="campo"><label for="cReg1">Nombres</label><input type="text" id="cReg1" value="${esc(nomReg(1))}" style="max-width:150px" placeholder="R1"><input type="text" id="cReg2" value="${esc(nomReg(2))}" style="max-width:150px" placeholder="R2"></div>` : ''}
          <p class="nota">Si volvés a cargar el archivo el mismo día (por ejemplo con turnos nuevos), no se pierde quién ya fue atendido. Si una persona tiene más de un turno en el mismo horario (aunque sean de registros distintos), aparece una sola vez con todos sus trámites; en horarios distintos, aparece una vez por horario. Al día siguiente la lista vieja se borra sola.</p></div>
        <div class="card pad"><h2>Actualizaciones</h2><p class="s">Versión instalada: <strong style="color:var(--ink)">${esc(V.version)}</strong>${u.disponible ? ` · disponible: <strong style="color:var(--ink)">${esc(u.version)}</strong>` : ' · está al día'}</p>
          <div style="display:flex;gap:10px;margin-top:14px;flex-wrap:wrap"><button class="btn" style="padding:0 18px" id="cBuscarUpd">Buscar actualizaciones</button>
          ${u.disponible ? '<button class="btn" style="background:#1E54B0;color:#fff;border:0;padding:0 18px" id="cAplicar">Actualizar ahora</button>' : ''}
          ${u.hayAnterior ? '<button class="btn rojo" style="padding:0 18px" id="cAnterior">Volver a la versión anterior</button>' : ''}</div>
          <div class="campo"><label>Canal</label><div class="segc"><button data-canal="" class="${!c.canal ? 'on' : ''}">General</button><button data-canal="prueba" class="${c.canal === 'prueba' ? 'on' : ''}">Prueba</button>${c.canal && c.canal !== 'prueba' ? `<button class="on">${esc(c.canal)}</button>` : ''}</div>
            <input type="text" id="cCanal" placeholder="Otro canal" style="max-width:130px" value=""></div>
          <p class="nota">Las actualizaciones se bajan de internet. Los datos de los turnos nunca salen de la oficina. En el canal <strong>Prueba</strong> las versiones nuevas llegan antes que al resto.</p></div>
        ${urlInstalacion() ? `<div class="card pad" style="grid-column:1/-1;display:flex;align-items:center;gap:18px;flex-wrap:wrap"><div style="flex:1;min-width:260px"><h2>Instalación en esta PC</h2><p class="s">Si TeToca está en su carpeta, si se abre solo al prender la PC y si tiene permiso de red. También para desinstalarlo.</p></div>
          <div style="display:flex;gap:10px"><a class="btn" style="padding:0 18px;text-decoration:none;display:inline-flex;align-items:center" href="${urlInstalacion()}">Instalación de esta PC…</a></div></div>` : ''}
        <div class="card pad" style="grid-column:1/-1"><h2>Tema del televisor</h2><p class="s">Cómo se ve la pantalla de la sala. Se cambia en el momento, sin reconectar el TV.</p>
          <div class="temas">${TEMAS_TV.map(([id, n, d, bg, top, card, txt, acc, pan]) => `<button class="tema ${(c.temaTV || 'noche') === id ? 'on' : ''}" data-temtv="${id}">
            <div class="mini" style="background:${bg};${id === 'contraste' ? 'outline:1px solid #FFE600;outline-offset:-1px' : ''}"><div class="t" style="background:${top};border-bottom:1px solid ${id === 'albiceleste' ? '#74ACDF' : 'rgba(128,128,128,.25)'}"></div>
              <div class="c"><div class="a" style="background:${card};border:1px solid rgba(128,128,128,.25)"><i style="background:${txt};width:80%"></i><i style="background:${acc};width:45%"></i></div><div class="l" style="background:${pan};border:1px solid rgba(128,128,128,.25)"></div></div></div>
            <b>${n}${id === 'noche' ? ' <span style="font-weight:600;color:var(--muted);font-size:12px">(actual)</span>' : ''}</b><small>${d}</small></button>`).join('')}</div></div>
        <div class="card pad" style="grid-column:1/-1"><h2>Televisor</h2><p class="s">Si el TV se conecta pero no muestra la pantalla de sala, probá el otro receptor. Los dos son gratuitos.</p>
          <div class="campo"><label>Receptor</label><div class="segc"><button data-rec="dashcast" class="${(c.tv.receptor || 'dashcast') === 'dashcast' ? 'on' : ''}">DashCast (recomendado)</button><button data-rec="urlcast" class="${c.tv.receptor === 'urlcast' ? 'on' : ''}">URL Cast Receiver</button></div>
          <a href="/tv" target="_blank" class="chico" style="text-decoration:none">Abrir pantalla de sala en el navegador</a></div>
          <div class="campo"><label>Sonido en el TV</label><button class="chico" id="cProbarTV">Probar sonido en el TV</button>
            <span id="cTVSon" style="font-size:13px;line-height:1.4">${resultadoSonidoTV()}</span></div></div>
      </div></div>`;
    document.querySelectorAll('[data-play]').forEach((b) => { b.onclick = () => { const id = b.dataset.play; sonando = id; const d = T.tocar(id); vistaConfig(); setTimeout(() => { sonando = null; if (vista === 'config' && !enEdicion()) vistaConfig(); }, d * 1000); }; });
    document.querySelectorAll('[data-elegir]').forEach((b) => { b.onclick = () => { T.tocar(b.dataset.elegir); api('/api/config', { sonido: b.dataset.elegir }); }; });
    document.querySelectorAll('[data-rol]').forEach((b) => { b.onclick = () => api('/api/pc', { pc: PC, rol: b.dataset.rol }); });
    document.querySelectorAll('[data-tema]').forEach((b) => { b.onclick = () => { tema = b.dataset.tema; guardarLocal('tetoca-tema', tema); render(); }; });
    document.querySelectorAll('[data-estilo]').forEach((b) => { b.onclick = () => { estilo = b.dataset.estilo; guardarLocal('tetoca-estilo', estilo); render(); }; });
    document.querySelectorAll('[data-temtv]').forEach((b) => { b.onclick = () => api('/api/config', { temaTV: b.dataset.temtv }); });
    document.querySelectorAll('[data-rec]').forEach((b) => { b.onclick = () => api('/api/tv/receptor', { receptor: b.dataset.rec }); });
    $('cProbarTV').onclick = () => { esperandoPrueba = Date.now(); api('/api/tv/probar-sonido', {}); vistaConfig(); };
    $('cBox').onchange = () => api('/api/pc', { pc: PC, box: $('cBox').value.trim() });
    $('cMostrar').onclick = () => api('/api/pc', { pc: PC, mostrarBox: !yo.mostrarBox });
    $('cOrg').onchange = () => api('/api/config', { organismo: $('cOrg').value.trim() });
    $('cOfi').onchange = () => api('/api/config', { oficina: $('cOfi').value.trim() });
    $('cSec').onchange = () => api('/api/config', { seccional: $('cSec').value.trim() });
    $('cUlt').onchange = () => api('/api/config', { ultimos: +$('cUlt').value });
    $('cAusN').onchange = () => api('/api/config', { ausentes: +$('cAusN').value });
    $('cLogo').onclick = () => $('archivoLogo').click();
    if ($('cSinLogo')) $('cSinLogo').onclick = () => api('/api/logo', undefined, 'DELETE');
    $('cCargar').onclick = () => cargarTurnos();
    $('cManual').onclick = agregarManual;
    document.querySelectorAll('[data-canal]').forEach((b) => { b.onclick = () => api('/api/config', { canal: b.dataset.canal }).then((r) => r && toast('Canal: ' + (b.dataset.canal || 'general') + '. Buscando actualizaciones…')); });
    $('cCanal').onchange = () => { const v = $('cCanal').value.trim().toLowerCase(); if (!/^[a-z0-9-]{1,30}$/.test(v)) { toast('El canal va en minúsculas, sin espacios (ej: tandil).', true); return; } api('/api/config', { canal: v }).then((r) => r && toast('Canal: ' + v + '.')); };
    document.querySelectorAll('[data-nreg]').forEach((b) => { b.onclick = () => api('/api/config', { registros: b.dataset.nreg === '2' ? [regs()[0] || 'R1', regs()[1] || 'R2'] : [] }); });
    const guardarRegs = () => api('/api/config', { registros: [$('cReg1').value.trim() || 'R1', $('cReg2').value.trim() || 'R2'] });
    if ($('cReg1')) { $('cReg1').onchange = guardarRegs; $('cReg2').onchange = guardarRegs; }
    $('cBuscarUpd').onclick = () => api('/api/update/buscar', undefined, 'GET').then((r) => r && toast(r.disponible ? 'Hay una versión nueva: ' + r.version : 'Ya tenés la última versión.'));
    if ($('cAplicar')) $('cAplicar').onclick = () => confirmar('Actualizar TeToca', 'El sistema se reinicia en unos segundos y todas las pantallas se recargan solas.', 'Actualizar ahora', () => api('/api/update/aplicar', {}));
    if ($('cAnterior')) $('cAnterior').onclick = () => confirmar('Volver a la versión anterior', 'Se vuelve a la versión que estaba antes de la última actualización. El sistema se reinicia en unos segundos.', 'Volver', () => api('/api/update/anterior', {}), 'rojo');
  }
  const enEdicion = () => document.activeElement && ['INPUT', 'SELECT', 'TEXTAREA'].includes(document.activeElement.tagName);

  // ---------- navegación y render ----------
  function irA(v) { vista = v; montada = null; render(); }
  function render() {
    if (!V) return;
    if (!vista || vista === 'rol') vista = miConf().rol || 'rol';
    barra();
    avisos();
    panelTV();
    if (vista === 'config') {
      if (montada !== 'config' || !enEdicion()) { vistaConfig(); montada = 'config'; }
      return;
    }
    if (vista === 'rol') { if (montada !== 'rol') { vistaRol(); montada = 'rol'; } return; }
    if (vista === 'mesa') { if (montada !== 'mesa') { montarMesa(); montada = 'mesa'; } actualizarMesa(); }
    if (vista === 'caja') { if (montada !== 'caja') { montarCaja(); montada = 'caja'; } actualizarCaja(); }
  }

  // ---------- archivos ----------
  $('archivo').onchange = async (e) => {
    const f = e.target.files[0]; e.target.value = '';
    if (!f) return;
    const fd = new FormData(); fd.append('archivo', f); fd.append('registro', cargaReg);
    try {
      const r = await fetch('/api/cargar', { method: 'POST', body: fd });
      const j = await r.json();
      if (!r.ok) { toast(j.error || 'No se pudo cargar el archivo.', true); return; }
      toast('Listo: se cargaron ' + j.cantidad + ' turnos' + (dosReg() ? ' de ' + nomReg(cargaReg) : '') + '.' + (j.aviso ? ' ' + j.aviso : ''), !!j.aviso);
      if (dosReg()) setTimeout(() => { if ($('modal').innerHTML) modalCarga(); }, 700);
    } catch (err) { toast('No hay conexión con la PC central.', true); }
  };
  $('archivoLogo').onchange = async (e) => {
    const f = e.target.files[0]; e.target.value = '';
    if (!f) return;
    const fd = new FormData(); fd.append('logo', f);
    const r = await fetch('/api/logo', { method: 'POST', body: fd });
    if (r.ok) toast('Logo actualizado.'); else toast('No se pudo subir el logo.', true);
  };

  // ---------- teclado ----------
  document.addEventListener('keydown', (e) => {
    if (enEdicion() || $('modal').innerHTML) return;
    if (e.code === 'Space') { e.preventDefault(); siguiente(); }
    else if (e.key === 'r' || e.key === 'R') { if (miActual()) accion('rellamar'); }
  });
  document.addEventListener('click', (e) => {
    if (popTV && !e.target.closest('.pop') && !e.target.closest('#bTV')) { popTV = false; render(); }
  });

  // ---------- conexión en tiempo real ----------
  T.conectar('/api/eventos?pc=' + encodeURIComponent(PC), (v) => {
    if (versionInicial && v.version !== versionInicial) { location.reload(); return; }
    versionInicial = v.version;
    V = v;
    $('desconectado').style.display = 'none';
    render();
  }, (ok, fallas) => {
    if (ok) return;
    if (fallas > 2) $('desconectado').style.display = 'flex';
    if (ES_PUESTO && fallas > 8) location.href = 'http://127.0.0.1:8767/';
  });
  setInterval(() => { if (V) barra(); }, 15000);
})();
