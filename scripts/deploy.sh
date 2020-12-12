#!/bin/sh -x

# Unpack archive.
rm -rf wtf
tar zxvf wtfd.tar.gz

s# Copy configuration & credentials.
chown wtf:wtf wtf/wtfd.conf && chmod 0644 wtf/wtfd.conf && cp wtf/wtfd.conf /etc/wtf/

# Set binary permissions.
chown wtf:wtf wtf/wtfd && chmod 0755 wtf/wtfd
setcap cap_net_bind_service=+ep wtf/wtfd

# Restart service.
service wtfd stop
mv wtf/wtfd /usr/local/bin/wtfd
service wtfd start
