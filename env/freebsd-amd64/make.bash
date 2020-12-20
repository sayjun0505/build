#!/bin/bash
# Copyright 2015 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Builds FreeBSD image based on raw disk images provided by FreeBSD.org
# This script boots the image once, side-loads GCE Go builder configuration via
# an ISO mounted as the CD-ROM, and customizes the system before powering down.
# SSH is enabled, and a user gopher, password gopher, is created.

# Only tested on Ubuntu 20.04.
# Requires packages: qemu-system-x86 qemu-img expect genisoimage

set -e

function download_image() {
  local img_dir=releases
  [ ${IS_SNAPSHOT:-0} -eq 1 ] && img_dir=snapshots
  local url=ftp://ftp.freebsd.org/pub/FreeBSD/${img_dir}/VM-IMAGES/${VERSION:?}/amd64/Latest
  local img_filename=FreeBSD-${VERSION:?}-amd64${VERSION_TRAILER}.raw.xz
  curl -O ${url}/${img_filename}
  echo "${SHA256}  ${img_filename}" | sha256sum -c -
  xz -d FreeBSD-${VERSION:?}-amd64${VERSION_TRAILER}.raw.xz
}

case $1 in
9.3)
  readonly VERSION=9.3-RELEASE
  readonly VERSION_TRAILER="-20140711-r268512"
  readonly SHA256=4737218995ae056207c68f3105c0fbe655c32e8b76d2160ebfb1bba56dd5196f
;;

10.3)
  readonly VERSION=10.3-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=1d710ba643bf6a8ce5bff5a9d69b1657ccff83dd1f2df711d9b4e02f9aab7d06
;;
10.4)
  readonly VERSION=10.4-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=8d1ff92e74a70f1ec039a465467f19abd7892331403ef1d4952d271adddab625
;;
11.0)
  readonly VERSION=11.0-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=f9f7fcac1acfe210979a72e0642a70fcf9c9381cc1884e966eac8381c724158c
  ;;
11.1)
  readonly VERSION=11.1
  readonly VERSION_TRAILER=
  readonly SHA256=233c6b269a29c1ce38bb4eb861251d1c74643846c1de937b8e31cc0316632bc0
;;
11.2)
  readonly VERSION=11.2-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=d8638aecbb13bdc891e17187f3932fe477f5655846bdaad8fecd60614de9312c
;;
11.3)
  readonly VERSION=11.3-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=e5f7fb12b828f0af7edf9464a08e51effef05ca9eb5fb52dba6d23a3c7a39223
;;
11.4)
  readonly VERSION=11.4-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=53a9db4dfd9c964d487d9f928754c964e2c3610c579c7f3558c745a75fa430f0
;;
12.0)
  readonly VERSION=12.0-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=9eb70a552f5395819904ed452a02e5805743459dbb1912ebafe4c9ae5de5eb53
;;
12.1)
  readonly VERSION=12.1-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=3750767f042ebf47a1e8e122b67d9cd48ec3cd2a4a60b724e64c4ff6ba33653a
;;
12.2)
  readonly VERSION=12.2-RELEASE
  readonly VERSION_TRAILER=
  readonly SHA256=0f8593382b6833658c6f6be532d4ffbedde7b75504452e27d912a0183f72ab56
;;
*)
  echo "Usage: $0 <version>"
  echo " version - FreeBSD version to build. Valid choices: 9.3 10.3 10.4 11.0 11.1 11.2 11.3 11.4 12.0 12.1 12.2"
  exit 1
esac

function cleanup() {
	rm -rf iso \
		*.iso \
		*.raw \
		*.qcow2
}

trap cleanup EXIT

if ! [ -e FreeBSD-${VERSION:?}-amd64.raw ]; then
  download_image
fi

qemu-img create -f qcow2 -b FreeBSD-${VERSION:?}-amd64${VERSION_TRAILER}.raw disk.qcow2 8G

mkdir -p iso/boot iso/etc iso/usr/local/etc/rc.d
cp loader.conf iso/boot
cp rc.conf iso/etc
cp sysctl.conf iso/etc
cp buildlet iso/usr/local/etc/rc.d

cat >iso/install.sh <<'EOF'
#!/bin/sh
set -x

mkdir -p /usr/local/etc/rc.d/
cp /mnt/usr/local/etc/rc.d/buildlet /usr/local/etc/rc.d/buildlet
chmod +x /usr/local/etc/rc.d/buildlet
cp /mnt/boot/loader.conf /boot/loader.conf
cp /mnt/etc/rc.conf /etc/rc.conf
cat /mnt/etc/sysctl.conf >> /etc/sysctl.conf
adduser -f - <<ADDUSEREOF
gopher::::::Gopher Gopherson::/bin/sh:gopher
ADDUSEREOF
pw user mod gopher -G wheel

# Enable serial console early in boot process.
echo '-h' > /boot.conf
EOF

genisoimage -r -o config.iso iso/
# TODO(wathiede): remove sleep
sleep 2

env DOWNLOAD_UPDATES=$((1-IS_SNAPSHOT)) expect <<'EOF'
set prompt "root@.*:~ #[ ]"
set timeout -1
set send_human {.1 .3 1 .05 2}

spawn qemu-system-x86_64 -machine graphics=off -display none -serial stdio \
 -fw_cfg name=opt/etc/sercon-port,string=0x3F8 \
 -m 1G -drive if=virtio,file=disk.qcow2,format=qcow2,cache=none -cdrom config.iso -net nic,model=virtio -net user
set qemu_pid $spawn_id

# boot with serial console enabled
expect -ex "Welcome to FreeBSD"
expect -re "Autoboot in \[0-9\]\+ seconds"
sleep 1
send -h "3" ;# escape to bootloader prompt
expect -ex "Type '?' for a list of commands, 'help' for more detailed help."
expect -ex "OK "
send -h "set console=\"comconsole\"\n"
expect -ex "OK "
send -h "boot\n"

# wait for login prompt
set timeout 180
expect {
    "\nlogin: " {
        send "root\n"
        sleep 1
    }
    timeout     { exit 2 }
}

expect -re $prompt
sleep 1
send "mount_cd9660 /dev/cd0 /mnt\nsh /mnt/install.sh\n"

expect -re $prompt
sleep 1

# generate SSH keys
send "service sshd keygen\n"
expect -re "Generating .+ host key."
sleep 1

expect -re $prompt
sleep 1
set timeout -1
# download updates
if {$::env(DOWNLOAD_UPDATES)} {
    send "env PAGER=cat freebsd-update fetch --not-running-from-cron\n"

    expect {
        "The following files will be updated as part of updating to" {
            sleep 2
            expect -re $prompt
            send "freebsd-update install\n"
            expect "Installing updates..."
            expect "done."
            sleep 1
            send "\n"
        }

        "No updates needed to update system to" {
            sleep 1
            send "\n"
        }

        "No mirrors remaining, giving up." { exit 3 }
    }
} else {
    puts "skipping updates"
    send "\n"
}

expect -re $prompt
sleep 1
send "pkg install -y bash curl git gdb\n"

expect -re $prompt
send "sync\n"

expect -re $prompt
send "poweroff\n"
expect "All buffers synced."

wait -i $qemu_pid
EOF

# Create Compute Engine disk image.
IMAGE=freebsd-amd64-${VERSION/-RELEASE/}.tar.gz
readonly IMAGE=${IMAGE/\./}
echo "Archiving disk.raw as ${IMAGE:?}... (this may take a while)"
qemu-img convert -f qcow2 -O raw -t none -T none disk.qcow2 disk.raw
tar -Szcf ${IMAGE:?} disk.raw

echo "Done. GCE image is ${IMAGE:?}"
