(function() {
  if (typeof WebSocket === 'undefined') return
  
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const ws = new WebSocket(protocol + '//' + window.location.host + '/ws')
  
  ws.onopen = function() {
    console.log('🔄 Live reload connected')
  }
  
  ws.onmessage = function(event) {
    if (event.data === 'reload') {
      console.log('🔄 Live reload triggered')
      window.location.reload()
    }
  }
  
  ws.onclose = function() {
    console.log('🔄 Live reload disconnected')
    // Attempt to reconnect after 1 second
    setTimeout(function() {
      if (ws.readyState === WebSocket.CLOSED) {
        location.reload()
      }
    }, 1000)
  }
  
  ws.onerror = function(error) {
    console.log('🔄 Live reload error:', error)
  }
})()