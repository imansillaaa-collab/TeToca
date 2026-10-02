#!/usr/bin/env python3
"""Genera los sonidos de llamado como archivos MP3 (web/sonidos/*.mp3).

Grabados de antemano suenan igual en todos lados (en el Chromecast, generarlos en
el momento sonaba "robótico"). Uso: python3 build/sonidos.py  (requiere numpy y ffmpeg)

Cómo están hechos para que se escuchen fuerte y limpio en el parlante chico de un TV:
- Sin compresores ni limitadores: aplastaban el sonido y metían un "click" al
  principio de cada nota (sonaba raro).
- Sin reverberación de ruido (ensuciaba el sonido).
- Notas en el rango medio (500-1600 Hz) con armónicas 2 y 3, que es donde los
  parlantes de TV rinden y el oído es más sensible. Por eso se escuchan sin graves.
- Notas que duran (decaen despacio), así llevan más energía con el mismo pico.
- Un silencio corto al principio para que el TV no se "coma" el arranque.
- Volumen al máximo posible sin saturar (pico a -1 dB después de codificar).
"""
import os, subprocess, wave
import numpy as np

SR = 44100
INICIO = 0.15   # silencio inicial
OUT = os.path.join(os.path.dirname(__file__), '..', 'web', 'sonidos')

# timbre: (múltiplo de la frecuencia, amplitud, qué tan rápido se apaga esa armónica)
CAMPANA = ((1, 1.0, 0.0), (2, 0.45, 1.2), (3, 0.22, 2.0), (4.2, 0.08, 4.0))
TIMBRE = ((1, 1.0, 0.0), (2, 0.40, 1.0), (3, 0.18, 2.0))
MARIMBA = ((1, 1.0, 0.0), (4, 0.30, 6.0), (2, 0.25, 2.0))


def nota(buf, t, f, dur, vol=1.0, timbre=TIMBRE, sosten=1.4, forma='seno'):
    n = int(dur * SR)
    i0 = int((INICIO + t) * SR)
    x = np.arange(n) / SR
    env = np.exp(-x * (sosten / dur))                    # se apaga despacio
    env *= np.clip(x / 0.012, 0, 1) ** 2                 # ataque suave (sin click)
    env *= np.clip((dur - x) / 0.06, 0, 1)               # cierre suave
    s = np.zeros(n)
    for mult, amp, apaga in timbre:
        fase = 2 * np.pi * f * mult * x
        if forma == 'cuadrada':
            w = np.sin(fase) + np.sin(3 * fase) / 3 + np.sin(5 * fase) / 5
        else:
            w = np.sin(fase)
        s += amp * w * np.exp(-x * apaga)
    end = min(len(buf), i0 + n)
    buf[i0:end] += vol * (s * env)[: end - i0]


def guardar(nombre, buf):
    fade = int(0.08 * SR)
    buf[-fade:] *= np.linspace(1, 0, fade)
    buf = buf / np.abs(buf).max() * 0.85
    wav = os.path.join(OUT, nombre + '.wav')
    mp3 = os.path.join(OUT, nombre + '.mp3')
    with wave.open(wav, 'wb') as w:
        w.setnchannels(1); w.setsampwidth(2); w.setframerate(SR)
        w.writeframes((buf * 32767).astype('<i2').tobytes())
    subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', wav, '-ar', '44100', '-ac', '2',
                    '-codec:a', 'libmp3lame', '-b:a', '192k', mp3], check=True)
    # ajuste fino: que el pico del MP3 quede en -1 dB (lo más fuerte sin saturar)
    pico = decodificar_pico(mp3)
    g = 10 ** (-1.0 / 20) / pico
    subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', wav, '-af', 'volume=%.4f' % g, '-ar', '44100', '-ac', '2',
                    '-codec:a', 'libmp3lame', '-b:a', '192k', mp3], check=True)
    os.remove(wav)


def decodificar_pico(mp3):
    r = subprocess.run(['ffmpeg', '-loglevel', 'error', '-i', mp3, '-f', 's16le', '-ac', '1', '-'], capture_output=True, check=True)
    a = np.frombuffer(r.stdout, dtype='<i2').astype(float) / 32768
    return np.abs(a).max()


def vacio(seg):
    return np.zeros(int((INICIO + seg) * SR))


os.makedirs(OUT, exist_ok=True)

# Ding-dong: dos notas que bajan (Mi - Do), una octava arriba de lo habitual para que el TV las rinda
b = vacio(2.4); nota(b, 0.0, 1319, 1.1, 1.0, CAMPANA, 1.6); nota(b, 0.5, 1047, 1.9, 0.8, CAMPANA, 1.4); guardar('dingdong', b)
b = vacio(2.5); nota(b, 0.0, 1175, 2.4, 1.0, CAMPANA, 1.8); guardar('campana', b)
b = vacio(2.3); [nota(b, i * 0.3, f, 0.9 if i < 2 else 1.5, 1.0, TIMBRE, 1.2) for i, f in enumerate((784, 988, 1175))]; guardar('tres', b)
b = vacio(1.7); [nota(b, i * 0.17, f, 0.55 if i < 2 else 1.0, 1.0, MARIMBA, 2.2) for i, f in enumerate((784, 988, 1175))]; guardar('marimba', b)
b = vacio(1.0); nota(b, 0.0, 1320, 0.22, 1.0, TIMBRE, 0.5); nota(b, 0.32, 1320, 0.3, 1.0, TIMBRE, 0.5); guardar('digital', b)
b = vacio(3.3); [nota(b, i * 0.45, f, 1.3, 1.0, CAMPANA, 1.4) for i, f in enumerate((1175, 988, 1047, 784))]; guardar('carillon', b)
print('listo:', sorted(os.listdir(OUT)))
