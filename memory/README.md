# Demonstrate process address space

Useful commands

* `getconf PAGE_SIZE`
* `cat /proc/meminfo`
* `grep MemTotal /proc/meminfo | cut -d' ' -f2- | numfmt --to=iec-i --suffix=B --format="%.2f"`
* `free -mht`
* `swapon -s`
