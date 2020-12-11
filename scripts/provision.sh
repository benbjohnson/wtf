#!/bin/sh -x

lineinfile() {
	if grep -q "$2" "$1"; then
		sed -i 's/'"$2"'/'"$3"'/' "$1"
	else
		echo "$3" >> "$1"
	fi
}

# Print each line.
set -o xtrace

# Set frontend.
export DEBIAN_FRONTEND=noninteractive

# Create wheel group.
groupadd wheel
echo "%wheel ALL=(ALL) NOPASSWD: ALL" >> "/etc/sudoers" 
visudo -cf /etc/sudoers

# Create benbjohnson user with login key.
useradd -m -G wheel -s /bin/bash benbjohnson
mkdir -p /home/benbjohnson/.ssh
chown benbjohnson:benbjohnson /home/benbjohnson/.ssh && chmod 700 /home/benbjohnson/.ssh
echo "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCpezYgoL33WgiS73KUHyvcsW5UZwp2SKV65QH5bdgZKxUDTSJKkzSSI+qPcueg8FLKYtKdZ2HBn+uYXzHabqRpmtw66Us2bJY0iitzG3V1Szb2RXZNPaf/eKWULUx55XaKZCVg8/viifAMGeC+SI6BdUQFD3LXpzGkRWWfKoqpZGD92400ORcmhOBW/a3afA34L3y+Z6O8LQ57RAHQxwKeTyWc/rgHoLvlyx3H3bV4kGAITCHJDSX+OSvyrwsx9yoi2CHiYBCt5K+gZJMV2P99uro4pyQF+6baDUtQJOW7sT2+2HTHNV8xLyutMDY6w8BQZdj/Z5+vRF0RUdBPi10q8xgb0N3fGkdF+c5W83K9+WwjPtQiNAwIGwrXzaQYsEm6ejkrkJQ6ZilX+cvDya0n42XxCooPRxNxCd9Wqq1o3aTR4ophFA8O1Dnei8RJo3mVuuVa7d8tXiVuTH/cYRYWsty8R/ueJ6Ipgwng4WnChG28beIhP3h9xMz4wi7BBRcazIk9nT/o5o/rcAd7TW+XHmyMdg75zKFBlEAKYWpbkyKesTaE5Ck1gYRmPIiXCcrzl7fg5+q25xFSHXFG4G/v3IpRFgK7AAOVwvJdsPP9m79Zj1g8FdZ/Tzr3yijy+++Y/Zh96O+z0+slkt77aj/NHGoa28jwgyP+oD4Qv7WMww== benbjohnson@yahoo.com" > /home/benbjohnson/.ssh/authorized_keys
chown benbjohnson:benbjohnson /home/benbjohnson/.ssh/authorized_keys && chmod 600 /home/benbjohnson/.ssh/authorized_keys

# Install and update packages.
apt update && apt upgrade -y
apt install -y ufw unattended-upgrades

# Set periodic upgrade settings.
cat <<EOF > /etc/apt/apt.conf.d/10periodic
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
APT::Periodic::Unattended-Upgrade "1";
EOF

# Create user.
useradd -s /sbin/nologin wtf

# Configure & enable firewall.
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow http
ufw allow https
ufw --force enable

# Disable root access & password authentication.
lineinfile "/etc/ssh/sshd_config" "^PermitRootLogin .*" "PermitRootLogin no"
lineinfile "/etc/ssh/sshd_config" "^PasswordAuthentication .*" "PasswordAuthentication no"
service sshd restart
