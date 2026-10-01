#!/usr/bin/env python3
"""Genera los sonidos de llamado como archivos MP3 (web/sonidos/*.mp3).

Antes se generaban en el momento con el navegador, pero en algunos televisores
(Chromecast) sonaban entrecortados o "robóticos". Grabados de antemano suenan
igual en todos lados. Uso: python3 build/sonidos.py  (requiere numpy y ffmpeg)
"""
import os, subprocess, wave
import numpy as np

SR = 44100
OUT = os.path.join(os.path.dirname(__file__), '..', 'web', 'sonidos')

def nota(buf, t, f, dur, vol, arm=((1, 1),), forma='sine', ataque=0.008):
    n = int(dur * SR)
    i0 = int(t * SR)
    x = np.arange(n) / SR
    env = np.exp(-x * (3.2 / dur))                      # se apaga natural
    env *= np.minimum(1, x / ataque)                    # ataque suave, sin "click"
    env *= np.minimum(1, (dur - x) / 0.03).clip(0, 1)   # cierre suave
    s = np.zeros(n)
    for mult, amp in arm:
        fase = 2 * np.pi * f * mult * x
        if forma == 'square':
            # "cuadrada" suave: pocas armónicas, sin el zumbido áspero
            w = sum(np.sin(fase * k) / k for k in (1, 3, 5))
        else:
            w = np.sin(fase)
        # las armónicas altas se apagan antes, como en una campana real
        s += amp * w * np.exp(-x * (mult - 1) * 2.5)
    end = min(len(buf), i0 + n)
    buf[i0:end] += vol * (s * env)[: end - i0]

def sala(buf):
    # un poquito de reverberación para que no suene "seco" en la sala
    ir_n = int(0.35 * SR)
    rng = np.random.default_rng(7)
    ir = rng.standard_normal(ir_n) * np.exp(-np.arange(ir_n) / SR * 14)
    ir[0] = 0
    ir /= np.abs(ir).sum()
    wet = np.convolve(buf, ir)[: len(buf)]
    return buf + 0.9 * wet

def comprimir(buf, umbral=0.06, ratio=8.0):
    # compresor suave: sube el volumen percibido sin distorsionar
    # (los parlantes de los televisores son chicos y la sala tiene ruido)
    env = np.abs(buf)
    a, r = np.exp(-1 / (0.004 * SR)), np.exp(-1 / (0.25 * SR))
    seg = np.empty_like(env)
    e = 0.0
    for i, v in enumerate(env):
        e = (a if v > e else r) * e + (1 - (a if v > e else r)) * v
        seg[i] = e
    g = np.ones_like(seg)
    m = seg > umbral
    g[m] = (umbral + (seg[m] - umbral) / ratio) / seg[m]
    return buf * g

def guardar(nombre, buf):
    buf = sala(buf)
    buf = buf / np.abs(buf).max()
    buf = comprimir(buf)
    buf = buf / np.abs(buf).max() * 0.97                 # bien fuerte, sin llegar a saturar
    fade = int(0.05 * SR)
    buf[-fade:] *= np.linspace(1, 0, fade)
    wav = os.path.join(OUT, nombre + '.wav')
    with wave.open(wav, 'wb') as w:
        w.setnchannels(1); w.setsampwidth(2); w.setframerate(SR)
        w.writeframes((buf * 32767).astype('<i2').tobytes())
    mp3 = os.path.join(OUT, nombre + '.mp3')
    subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', wav,
                    # volumen alto y parejo entre los seis sonidos, con limitador para no saturar
                    '-af', 'acompressor=threshold=-24dB:ratio=6:attack=2:release=200:makeup=6,alimiter=limit=0.89:attack=1:release=50,loudnorm=I=-9:TP=-0.8:LRA=5',
                    '-ar', '44100', '-codec:a', 'libmp3lame', '-b:a', '128k', mp3], check=True)
    os.remove(wav)

def vacio(seg):
    return np.zeros(int(seg * SR))

os.makedirs(OUT, exist_ok=True)
CAMP = ((1, 1), (2.76, 0.35), (5.4, 0.15))
MAR = ((1, 1), (4, 0.25))

b = vacio(2.3); nota(b, 0.02, 659, 1.3, 1, ((1, 1), (2, 0.2))); nota(b, 0.47, 523, 1.7, 1, ((1, 1), (2, 0.2))); guardar('dingdong', b)
b = vacio(2.5); nota(b, 0.02, 880, 2.3, 1, CAMP); guardar('campana', b)
b = vacio(2.3); [nota(b, 0.02 + i * 0.28, f, 0.9 if i < 2 else 1.5, 1) for i, f in enumerate((523, 659, 784))]; guardar('tres', b)
b = vacio(1.6); [nota(b, 0.02 + i * 0.16, f, 0.5 if i < 2 else 0.9, 1, MAR) for i, f in enumerate((784, 988, 1175))]; guardar('marimba', b)
b = vacio(1.0); nota(b, 0.02, 1320, 0.18, 0.6, forma='square'); nota(b, 0.32, 1320, 0.18, 0.6, forma='square'); guardar('digital', b)
b = vacio(3.2); [nota(b, 0.02 + i * 0.42, f, 1.3, 1, ((1, 1), (2, 0.25), (3, 0.08))) for i, f in enumerate((784, 659, 698, 523))]; guardar('carillon', b)
print('listo:', sorted(os.listdir(OUT)))
