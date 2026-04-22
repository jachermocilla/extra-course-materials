#!/bin/bash
# ntp_demo.sh - Useful NTP tasks on Linux

echo "========================================"
echo "         NTP DEMO FOR LINUX"
echo "========================================"

# ── 1. Check if systemd-timesyncd or ntpd is running ──────────────────────────
echo ""
echo "[ 1 ] TIME SYNC SERVICE STATUS"
echo "----------------------------------------"
if systemctl is-active --quiet systemd-timesyncd; then
    echo "systemd-timesyncd is RUNNING"
    timedatectl show-timesync --all 2>/dev/null || true
elif systemctl is-active --quiet ntpd; then
    echo "ntpd is RUNNING"
else
    echo "No active time sync service found"
fi

# ── 2. Current time and sync status ───────────────────────────────────────────
echo ""
echo "[ 2 ] CURRENT TIME & SYNC STATUS"
echo "----------------------------------------"
timedatectl status

# ── 3. Show NTP servers being used ────────────────────────────────────────────
echo ""
echo "[ 3 ] NTP SERVERS IN USE"
echo "----------------------------------------"
if command -v chronyc &>/dev/null; then
    echo "Using chrony:"
    chronyc sources -v
elif command -v ntpq &>/dev/null; then
    echo "Using ntpq:"
    ntpq -p
else
    echo "Using systemd-timesyncd:"
    grep -E "^NTP|^FallbackNTP" /etc/systemd/timesyncd.conf 2>/dev/null \
        || echo "  (default pool: ntp.ubuntu.com)"
    timedatectl show-timesync 2>/dev/null | grep -E "ServerName|ServerAddress"
fi

# ── 4. Current offset and delay from NTP server ───────────────────────────────
echo ""
echo "[ 4 ] OFFSET & DELAY"
echo "----------------------------------------"
if command -v chronyc &>/dev/null; then
    chronyc tracking | grep -E "System time|Last offset|RMS offset|Frequency|Root delay"
elif command -v ntpq &>/dev/null; then
    ntpq -c rv | tr ',' '\n' | grep -E "offset|delay|jitter"
else
    timedatectl show-timesync 2>/dev/null \
        | grep -E "Offset|Delay|Jitter|Leap"
fi

# ── 5. One-shot manual sync (dry run shown here) ──────────────────────────────
echo ""
echo "[ 5 ] MANUAL ONE-SHOT SYNC"
echo "----------------------------------------"
echo "Command to force immediate sync (requires root):"
echo "  sudo systemctl restart systemd-timesyncd"
echo "  # or with chrony:"
echo "  sudo chronyc makestep"
echo "  # or with ntpdate (legacy):"
echo "  sudo ntpdate -u pool.ntp.org"

# ── 6. Query a public NTP server directly ─────────────────────────────────────
echo ""
echo "[ 6 ] QUERY PUBLIC NTP SERVER DIRECTLY"
echo "----------------------------------------"
if command -v ntpdate &>/dev/null; then
    echo "Querying pool.ntp.org (no adjust, just display):"
    ntpdate -q pool.ntp.org 2>/dev/null | tail -1
elif command -v sntp &>/dev/null; then
    echo "Querying pool.ntp.org via sntp:"
    sntp -t 2 pool.ntp.org 2>/dev/null || echo "  (sntp query failed or timed out)"
else
    echo "ntpdate/sntp not installed."
    echo "Install with: sudo apt install ntpdate"
fi

# ── 7. Compute offset manually using the 4-timestamp method ───────────────────
echo ""
echo "[ 7 ] MANUAL 4-TIMESTAMP OFFSET ESTIMATE (nc trick)"
echo "----------------------------------------"
echo "Concept: simulate T1/T2/T3/T4 using date before and after a network call"

T1=$(date +%s%N)  # nanoseconds

# Simulate a network call (ping a time server, measure RTT)
PING_MS=$(ping -c 1 -W 2 pool.ntp.org 2>/dev/null \
    | grep "time=" | grep -oP "time=\K[0-9.]+")

T4=$(date +%s%N)

if [[ -n "$PING_MS" ]]; then
    RTT_NS=$(echo "$PING_MS * 1000000" | bc | cut -d. -f1)
    HALF_RTT_NS=$((RTT_NS / 2))
    echo "  Round-trip delay : ${PING_MS} ms"
    echo "  Estimated one-way: $(echo "scale=3; $PING_MS/2" | bc) ms"
    echo "  T1=$T1 ns"
    echo "  T4=$T4 ns"
    echo "  Elapsed (T4-T1)  : $(( (T4 - T1) / 1000000 )) ms"
else
    echo "  Could not reach pool.ntp.org (no network or ping blocked)"
fi

# ── 8. Show hardware clock vs system clock ────────────────────────────────────
echo ""
echo "[ 8 ] HARDWARE CLOCK vs SYSTEM CLOCK"
echo "----------------------------------------"
echo -n "  System clock (software) : "
date --utc "+%Y-%m-%d %H:%M:%S UTC"
echo -n "  Hardware clock (RTC)     : "
sudo hwclock --utc --show 2>/dev/null || echo "(requires root)"

# ── 9. Timezone info ──────────────────────────────────────────────────────────
echo ""
echo "[ 9 ] TIMEZONE"
echo "----------------------------------------"
timedatectl | grep -E "Time zone|Universal"

echo ""
echo "========================================"
echo "              DONE"
echo "========================================"
