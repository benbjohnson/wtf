default: generate

# Runs the ego templating generation tool whenever an HTML template changes.
generate: http/html/*.ego
	@ego ./http/html

# Removes all ego Go files from the http/html directory.
clean:
	@rm http/html/*.ego.go

.PHONY: default generate clean