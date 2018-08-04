import $ from 'jquery';
import Noty from 'noty';
import Account from './account.js';
import Engine from './engine.js';
import Mixin from './mixin.js';
import Ocean from './ocean.js';
import Order from './order.js';
import Asset from './asset.js';

function API(router, root, engine) {
  this.router = router;
  this.root = root;
  this.account = new Account(this);
  this.mixin = new Mixin(this);
  this.ocean = new Ocean(this);
  this.engine = new Engine(engine);
  this.order = new Order(this);
  this.asset = new Asset(this);
  this.Error404 = require('../404.html');
  this.ErrorGeneral = require('../error.html');
}

API.prototype = {
  request: function(method, path, params, callback) {
    const self = this;
    var body = JSON.stringify(params);
    var url = self.root + path;
    if (path.indexOf('https://') === 0) {
      url = path;
    }
    if (url.indexOf('https://api.mixin.one') === 0) {
      var uri = path.slice('https://api.mixin.one'.length);
      self.account.mixinToken(uri, function (resp) {
        if (resp.error) {
          return callback(error);
        }
        return self.send(resp.data.token, method, url, body, callback);
      });
    } else if (url.indexOf('https://events.ocean.one/orders') === 0) {
      self.account.oceanToken(function (resp) {
        if (resp.error) {
          return callback(error);
        }
        return self.send(resp.data.token, method, url, body, callback);
      });
    } else {
      var token = self.account.token(method, path, body);
      return self.send(token, method, url, body, callback);
    }
  },

  send: function (token, method, url, body, callback) {
    const self = this;
    $.ajax({
      type: method,
      url: url,
      contentType: "application/json",
      data: body,
      beforeSend: function(xhr) {
        xhr.setRequestHeader("Authorization", "Bearer " + token);
      },
      success: function(resp) {
        var consumed = false;
        if (typeof callback === 'function') {
          consumed = callback(resp);
        }
        if (!consumed && resp.error !== null && resp.error !== undefined) {
          self.error(resp);
        }
      },
      error: function(event) {
        self.error(event.responseJSON, callback);
      }
    });
  },

  error: function(resp, callback) {
    if (resp == null || resp == undefined || resp.error === null || resp.error === undefined) {
      resp = {error: { code: 0, description: 'unknown error' }};
    }

    var consumed = false;
    if (typeof callback === 'function') {
      consumed = callback(resp);
    }
    if (!consumed) {
      switch (resp.error.code) {
        case 401:
          this.account.clear();
          this.router.replace('/sessions/new');
          break;
        case 404:
          $('#layout-container').html(this.Error404());
          $('body').attr('class', 'error layout');
          this.router.updatePageLinks();
          break;
        default:
          if ($('#layout-container > .spinner-container').length === 1) {
            $('#layout-container').html(this.ErrorGeneral());
            $('body').attr('class', 'error layout');
            this.router.updatePageLinks();
          }
          this.notify('error', i18n.t('general.errors.' + resp.error.code));
          break;
      }
    }
  },

  notify: function(type, text) {
    new Noty({
      type: type,
      layout: 'top',
      theme: 'nest',
      text: text,
      timeout: 3000,
      progressBar: false,
      queue: 'api',
      killer: 'api',
      force: true,
      animation: {
        open: 'animated bounceInDown',
        close: 'animated slideOutUp noty'
      }
    }).show();
  }
};

export default API;
