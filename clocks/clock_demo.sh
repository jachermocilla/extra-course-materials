#!/bin/bash
# clock_demo.sh - Hardware vs Software clock demonstration on Linux

echo "========================================"
echo "   HARDWARE vs SOFTWARE CLOCK DEMO"
echo "========================================"

# ── 1. Software clock reads ───────────────────────────────────────────────────
echo ""
echo "[ 1 ] SOFTWARE CLOCK (kernel clock)"
echo "----------------------------------------"

echo -n "  date (local)       : "
date

echo -n "  date (UTC)         : "
date --utc

echo -n "  Unix timestamp (s) : "
date +%s

echo -n "  High-res (ns)      : "
date +%s%N

echo -n "  clock_gettime via  : "
python3 -c "import time; t=time.clock_gettime(time.CLOCK_REALTIME); print(f'{t:.9f} s')" 2>/dev/null \
    || echo "(python3 not available)"

echo -n "  CLOCK_MONOTONIC    : "
python3 -c "import time; t=time.clock_gettime(time.CLOCK_MONOTONIC); print(f'{t:.9f} s (uptime-based, never jumps)')" 2>/dev/null \
    || echo "(python3 not available)"

# ── 2. Hardware clock (RTC) reads ─────────────────────────────────────────────
echo ""
echo "[ 2 ] HARDWARE CLOCK (RTC)"
echo "----------------------------------------"

echo -n "  hwclock (local)    : "
sudo hwclock --show 2>/dev/null || echo "(requires root -- try: sudo ./clock_demo.sh)"

echo -n "  hwclock (UTC)      : "
sudo hwclock --show --utc 2>/dev/null || echo "(requires root)"

echo -n "  hwclock (verbose)  : "
echo ""
sudo hwclock --verbose 2>/dev/null | grep -E "Using|Time|Drift|adjust" \
    || echo "  (requires root)"

# ── 3. Side-by-side drift comparison ─────────────────────────────────────────
echo ""
echo "[ 3 ] SIDE-BY-SIDE: SOFTWARE vs HARDWARE"
echo "----------------------------------------"
SW=$(date --utc "+%Y-%m-%d %H:%M:%S")
HW=$(sudo hwclock --utc --show 2>/dev/null | cut -d' ' -f1-3 || echo "N/A (needs root)")
echo "  Software clock : $SW UTC"
echo "  Hardware clock : $HW"

# ── 4. Measure software clock resolution ─────────────────────────────────────
echo ""
echo "[ 4 ] CLOCK RESOLUTION"
echo "----------------------------------------"
python3 - <<'EOF' 2>/dev/null || echo "  (python3 not available)"
import time

clocks = {
    "CLOCK_REALTIME":           time.CLOCK_REALTIME,
    "CLOCK_MONOTONIC":          time.CLOCK_MONOTONIC,
    "CLOCK_MONOTONIC_RAW":      time.CLOCK_MONOTONIC_RAW,
    "CLOCK_PROCESS_CPUTIME_ID": time.CLOCK_PROCESS_CPUTIME_ID,
}

for name, clk in clocks.items():
    res = time.clock_getres(clk)
    print(f"  {name:<30} resolution = {res:.9f} s")
EOF

# ── 5. Timer interrupt tick rate ─────────────────────────────────────────────
echo ""
echo "[ 5 ] KERNEL TICK RATE (CONFIG_HZ)"
echo "----------------------------------------"
HZ=$(grep "^CONFIG_HZ=" /boot/config-$(uname -r) 2>/dev/null | cut -d= -f2)
if [[ -n "$HZ" ]]; then
    TICK_MS=$(echo "scale=3; 1000/$HZ" | bc)
    echo "  CONFIG_HZ  = $HZ Hz"
    echo "  Tick period= ${TICK_MS} ms per interrupt"
else
    echo "  CONFIG_HZ not found, trying /proc/timer_list..."
    grep "tick_sched_timer" /proc/timer_list 2>/dev/null | head -3 \
        || echo "  (not accessible)"
fi

# ── 6. System uptime (CLOCK_MONOTONIC source) ─────────────────────────────────
echo ""
echo "[ 6 ] UPTIME (CLOCK_MONOTONIC)"
echo "----------------------------------------"
echo -n "  uptime             : "; uptime -p
echo -n "  /proc/uptime       : "
read UP IDLE < /proc/uptime
printf "  up=%.2fs  idle=%.2fs\n" $UP $IDLE

# ── 7. Live drift: sample software clock 5 times ─────────────────────────────
echo ""
echo "[ 7 ] LIVE SOFTWARE CLOCK SAMPLES (5 x 0.2s apart)"
echo "----------------------------------------"
python3 - <<'EOF' 2>/dev/null || echo "  (python3 not available)"
import time
prev = None
for i in range(5):
    t = time.clock_gettime(time.CLOCK_REALTIME)
    if prev:
        delta = (t - prev) * 1000
        print(f"  sample {i+1}: {t:.6f}  Δ={delta:.3f} ms")
    else:
        print(f"  sample {i+1}: {t:.6f}")
    prev = t
    time.sleep(0.2)
EOF

# ── 8. Sync state from kernel ─────────────────────────────────────────────────
echo ""
echo "[ 8 ] KERNEL CLOCK SYNC STATE (/proc/)"
echo "----------------------------------------"
echo -n "  adjtimex state     : "
python3 -c "
import ctypes, os
CLOCK_REALTIME = 0
class Timex(ctypes.Structure):
    _fields_ = [('modes',ctypes.c_uint),('offset',ctypes.c_long),
                ('freq',ctypes.c_long),('maxerror',ctypes.c_long),
                ('esterror',ctypes.c_long),('status',ctypes.c_int),
                ('constant',ctypes.c_long),('precision',ctypes.c_long),
                ('tolerance',ctypes.c_long),('time_sec',ctypes.c_long),
                ('time_usec',ctypes.c_long),('tick',ctypes.c_long),
                ('ppsfreq',ctypes.c_long),('jitter',ctypes.c_long),
                ('shift',ctypes.c_int),('stabil',ctypes.c_long),
                ('jitcnt',ctypes.c_long),('calcnt',ctypes.c_long),
                ('errcnt',ctypes.c_long),('stbcnt',ctypes.c_long)]
tx = Timex()
libc = ctypes.CDLL('libc.so.6', use_errno=True)
ret = libc.adjtimex(ctypes.byref(tx))
states = {0:'OK',1:'INS',2:'DEL',3:'OOP',4:'WAIT',5:'ERROR'}
print(f'status={states.get(ret,ret)}  offset={tx.offset}us  freq={tx.freq}  maxerror={tx.maxerror}us')
" 2>/dev/null || echo "(python3 ctypes not available)"

# ── 9. RTC device info ────────────────────────────────────────────────────────
echo ""
echo "[ 9 ] RTC DEVICE INFO"
echo "----------------------------------------"
if [[ -e /dev/rtc0 ]]; then
    echo "  RTC device: /dev/rtc0 exists"
    sudo cat /proc/driver/rtc 2>/dev/null | grep -E "rtc_time|rtc_date|24hr|alrm" \
        || echo "  (requires root to read /proc/driver/rtc)"
else
    echo "  /dev/rtc0 not found"
fi

ls /sys/class/rtc/ 2>/dev/null && \
    cat /sys/class/rtc/rtc0/name 2>/dev/null | xargs -I{} echo "  RTC driver: {}"

# ── 10. Sync hardware clock to software clock ─────────────────────────────────
echo ""
echo "[ 10 ] SYNC COMMANDS (shown only, not executed)"
echo "----------------------------------------"
echo "  # Set hardware clock FROM system clock:"
echo "  sudo hwclock --systohc"
echo ""
echo "  # Set system clock FROM hardware clock:"
echo "  sudo hwclock --hctosys"
echo ""
echo "  # Set system clock manually:"
echo "  sudo date -s '2026-04-16 12:00:00'"
echo ""
echo "  # Force NTP resync:"
echo "  sudo systemctl restart systemd-timesyncd"

echo ""
echo "========================================"
echo "                DONE"
echo "========================================"
