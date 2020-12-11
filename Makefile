default: generate

# Runs the ego templating generation tool whenever an HTML template changes.
generate: http/html/*.ego
	@ego ./http/html

# Removes all ego Go files from the http/html directory.
clean:
	@rm http/html/*.ego.go

# Build wtfd binary for Linux & package into tarball.
dist: generate
	@rm -rf dist && mkdir -p dist/wtf
	@blackbox_cat scripts/etc/wtfd.conf.gpg > dist/wtf/wtfd.conf
	@cp scripts/etc/wtfd.service dist/wtf/wtfd.service
	@GOOS=linux GOARCH=amd64 go build -o dist/wtf/wtfd ./cmd/wtfd
	@tar czvf dist/wtfd.tar.gz -C dist wtfd
	@rm -rf dist/wtfd

# Packages the binary & systemd service definition and deploys to the server.
deploy: dist
	@scp dist/wtfdial.tar.gz wtfdial.com:
	@ssh wtfdial.com 'sudo bash -s' < scripts/deploy.sh

# Provision a new server.
provision:
	@ssh root@wtfdial.com 'sudo bash -s' < scripts/provision.sh

# Removes the third party theme from the file system.
remove-theme:
	@rm http/assets/css/theme.css

.PHONY: default generate clean remove-theme
