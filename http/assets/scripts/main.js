function connect() {
	const socket = new ReconnectingWebSocket((location.protocol == 'https:' ? 'wss:' : 'ws:') + '//' + location.host + '/events');
	socket.addEventListener('message', function (event) {
		const e = JSON.parse(event.data)
		console.log(e)

		switch (e.type) {
		case "dial:value_changed":
			document.querySelectorAll('.wtf-value[data-dial-id="'+e.payload.id+'"]').forEach(
				(node) => updateWTFValueNode(node, e.payload.value)
			)
			if (window.ondialvaluechanged !== undefined) {
				window.ondialvaluechanged(e.payload)
			}
			break;

		case "dial_membership:value_changed":
			document.querySelectorAll('.wtf-value[data-dial-membership-id="'+e.payload.id+'"]').forEach(
				(node) => updateWTFValueNode(node, e.payload.value)
			)
			if (window.ondialmembershipvaluechanged !== undefined) {
				window.ondialmembershipvaluechanged(e.payload)
			}
			break;
		}
	});
}

function updateWTFValueNode(node, value) {
	// Update text value.
	node.innerText = value

	// If this is a badge, update the color.
	if (node.classList.contains('wtf-badge')) {
		// Remove old color.
		node.classList.remove("badge-soft-success", "badge-soft-info", "badge-soft-warning", "badge-soft-danger")

		// Set new color based on value.
		if (value < 25) {
			node.classList.add("badge-soft-success")
		} else if (value < 50) {
			node.classList.add("badge-soft-info")
		} else if (value < 75) {
			node.classList.add("badge-soft-warning")
		} else {
			node.classList.add("badge-soft-danger")
		}
	}
}