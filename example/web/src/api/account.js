import forge from 'node-forge';
import moment from 'moment';
import KJUR from 'jsrsasign';
import uuid from 'uuid/v4';

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
    var ec = new KJUR.crypto.ECDSA({'curve': 'secp256r1'});
    var pub = ec.generateKeyPairHex().ecpubhex;
    var priv = KJUR.KEYUTIL.getPEM(ec, 'PKCS8PRV');

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
    var ec = new KJUR.crypto.ECDSA({'curve': 'secp256r1'});
    var pub = ec.generateKeyPairHex().ecpubhex;
    var priv = KJUR.KEYUTIL.getPEM(ec, 'PKCS8PRV');

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

    var uid = window.localStorage.getItem('uid');
    var sid = window.localStorage.getItem('sid');
    return this.sign(uid, sid, priv, method, uri, body);
  },

  sign: function (uid, sid, privateKey, method, uri, body) {
    if (typeof body !== 'string') { body = ""; }

    let expire = moment.utc().add(1, 'minutes').unix();
    let md = forge.md.sha256.create();
    md.update(method + uri + body);

    var oHeader = {alg: 'ES256', typ: 'JWT'};
    var oPayload = {
      uid: uid,
      sid: sid,
      iat: moment.utc().unix() ,
      exp: expire,
      jti: uuid(),
      sig: md.digest().toHex()
    };
    var sHeader = JSON.stringify(oHeader);
    var sPayload = JSON.stringify(oPayload);
    try {
      KJUR.KEYUTIL.getKey(privateKey);
    } catch {
      return "";
    }
    return KJUR.jws.JWS.sign("ES256", sHeader, sPayload, privateKey);
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
      return callback({data: {token: token}});
    }
    var params = {
      category: category,
      uri: uri
    };
    this.api.request('POST', '/tokens', params, function(resp) {
      if (resp.data) {
        window.localStorage.setItem(key, resp.data.token);
      }
      return callback(resp);
    });
  },

  clear: function () {
    var d = window.localStorage.getItem('market.default');
    window.localStorage.clear();
    if (d) {
      window.localStorage.setItem('market.default', d);
    }
  }
}

export default Account;
