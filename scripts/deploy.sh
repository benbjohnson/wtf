#!/bin/sh -x

# Unpack archive.
rm -rf wtf
tar zxvf wtfd.tar.gz

# Ensure var/lib directory exists.
mkdir -p /var/lib/wtfd && chown wtf:wtf /var/lib/wtfd && chmod 755 /var/lib/wtfd

# Copy configuration & credentials.
mkdir -p /etc/wtfd && chown wtf:wtf /etc/wtfd && chmod 0755 /etc/wtfd
chown wtf:wtf wtf/wtfd.conf && chmod 0644 wtf/wtfd.conf && cp wtf/wtfd.conf /etc/wtf/

# Set binary permissions.
chown wtf:wtf wtf/wtfd && chmod 0755 wtf/wtfd
setcap cap_net_bind_service=+ep wtf/wtfd

# Deploy service.
chown root:root wtf/wtfd.service && chmod 0644 wtf/wtfd.service
cp wtf/wtfd.service /etc/systemd/system/
systemctl enable wtfd.service

# Restart service.
service wtfd stop
mv wtf/wtfd /usr/local/bin/wtfd
service wtfd start
