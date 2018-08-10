import ReconnectingWebSocket from 'reconnecting-websocket';
import pako from 'pako';
import uuidv4 from 'uuid/v4';

function Engine(endpoint) {
  const self = this;
  self.handlers = {};
  self.ws = new ReconnectingWebSocket(endpoint, [], {
    maxReconnectionDelay: 5000,
    minReconnectionDelay: 1000,
    reconnectionDelayGrowFactor: 1.2,
    connectionTimeout: 4000,
    maxRetries: Infinity,
    debug: false
  });

  self.ws.addEventListener("message", (event) => {
    var fileReader = new FileReader();
    fileReader.onload = function() {
      var msg = pako.ungzip(new Uint8Array(this.result), { to: 'string' });
      self.handle(JSON.parse(msg));
    };
    fileReader.readAsArrayBuffer(event.data);
  });

  self.ws.addEventListener("open", (event) => {
    for (var i in self.handlers) {
      self.send(self.handlers[i].message);
    }
  });

  self.ws.addEventListener('close', () => self.ws._shouldReconnect && self.ws._connect());
}

Engine.prototype = {
  reset: function() {
    try {
      this.ws.close();
    } catch (e) {
      if (e instanceof DOMException) {
      } else {
        console.error(e);
      }
    }
  },

  send: function (msg) {
    try {
      this.ws.send(pako.gzip(JSON.stringify(msg)));
    } catch (e) {
      if (e instanceof DOMException) {
      } else {
        console.error(e);
      }
    }
  },

  handle: function (msg) {
    var handler = this.handlers[msg.data.market];
    if (handler) {
      handler.callback(msg);
    }
  },

  subscribe: function (market, callback) {
    var handler = this.handlers[market];
    if (handler) {
      this.unsubscribe(market);
    }
    var msg = {
      id: uuidv4().toLowerCase(),
      action: 'SUBSCRIBE_BOOK',
      params: { market: market }
    };
    this.send(msg);
    this.handlers[market] = {
      callback: callback,
      message: msg
    };
  },

  unsubscribe: function (market) {
    var handler = this.handlers[market];
    if (handler) {
      delete self.handlers[market];
      this.send({
        id: uuidv4().toLowerCase(),
        action: 'UNSUBSCRIBE_BOOK',
        params: { market: market }
      });
    }
  }
};

export default Engine;
