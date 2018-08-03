import forge from 'node-forge';
import moment from 'moment';
import jwt from 'jsonwebtoken';
import uuid from 'uuid/v4';
const EC = require('elliptic').ec;
const KeyEncoder = require('key-encoder');

function Account(api) {
  this.api = api;
}

Account.prototype = {
  newVerification: function (callback, params) {
    this.api.request('POST', '/verifications', params, function(resp) {
      return callback(resp);
    });
  },

  verifyVerification: function (callback, params) {
    this.api.request('POST', '/verifications/' + params['verification_id'], params, function(resp) {
      return callback(resp);
    });
  },

  createUser: function (callback, params) {
    var ec = new EC('p256');
    var key = ec.genKeyPair();
    var pub = key.getPublic('hex');
    var priv = key.getPrivate('hex');

    params['session_secret'] = '3059301306072a8648ce3d020106082a8648ce3d030107034200' + pub;
    this.api.request('POST', '/users', params, function(resp) {
      if (resp.data) {
        window.localStorage.setItem('token.example', priv);
        window.localStorage.setItem('uid', resp.data.user_id);
        window.localStorage.setItem('sid', resp.data.session_id);
      }
      return callback(resp);
    });
  },

  createSession: function (callback, params) {
    var ec = new EC('p256');
    var key = ec.genKeyPair();
    var pub = key.getPublic('hex');
    var priv = key.getPrivate('hex');

    params['session_secret'] = '3059301306072a8648ce3d020106082a8648ce3d030107034200' + pub;
    this.api.request('POST', '/sessions', params, function(resp) {
      if (resp.data) {
        window.localStorage.setItem('token.example', priv);
        window.localStorage.setItem('uid', resp.data.user_id);
        window.localStorage.setItem('sid', resp.data.session_id);
      }
      return callback(resp);
    });
  },

  me: function (callback) {
    const self = this;
    this.api.request('GET', '/me', undefined, function(resp) {
      if (typeof callback === 'function') {
        return callback(resp);
      }
    });
  },

  token: function (method, uri, body) {
    var priv = window.localStorage.getItem('token.example');
    if (!priv) {
      return "";
    }

    var encoderOptions = {
      curveParameters: [1, 2, 840, 10045, 3, 1, 7],
      privatePEMOptions: {label: 'EC PRIVATE KEY'},
      publicPEMOptions: {label: 'PUBLIC KEY'},
      curve: new EC('p256')
    };
    var keyEncoder = new KeyEncoder(encoderOptions);
    var pemPrivateKey = keyEncoder.encodePrivate(priv, 'raw', 'pem');

    var uid = window.localStorage.getItem('uid');
    var sid = window.localStorage.getItem('sid');
    return this.sign(uid, sid, pemPrivateKey, method, uri, body);
  },

  sign: function (uid, sid, privateKey, method, uri, body) {
    if (typeof body !== 'string') { body = ""; }

    let expire = moment.utc().add(1, 'minutes').unix();
    let md = forge.md.sha256.create();
    md.update(method + uri + body);
    let payload = {
      uid: uid,
      sid: sid,
      iat: moment.utc().unix() ,
      exp: expire,
      jti: uuid(),
      sig: md.digest().toHex()
    };
    return jwt.sign(payload, privateKey, { algorithm: 'ES256'});
  },

  oceanToken: function (callback) {
    this.externalToken("OCEAN", "", callback);
  },

  mixinToken: function (uri, callback) {
    this.externalToken("MIXIN", uri, callback);
  },

  externalToken: function (category, uri, callback) {
    var key = 'token.' + category.toLowerCase() + uri;
    var token = window.localStorage.getItem(key);
    if (token) {
      return callback(token);
    }
    var params = {
      category: category,
      uri: uri
    };
    this.api.request('POST', '/tokens', params, function(resp) {
      if (resp.data) {
        window.localStorage.setItem(key, resp.data.token);
        return callback(resp.data.token);
      }
    });
  },

  clear: function () {
    window.localStorage.clear();
  }
}

export default Account;
