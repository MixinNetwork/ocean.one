const EC = require('elliptic').ec;

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

  check: function (callback) {
    const self = this;
    this.api.request('GET', '/me', undefined, function(resp) {
      if (typeof callback === 'function') {
        return callback(resp);
      }
    });
  },

  token: function () {
    var priv = window.localStorage.getItem('token.example');
    if (!priv) {
      return "";
    }
    var ec = new EC('p256');
    var key = ec.keyFromPrivate(priv);
    var oHeader = {alg: 'ES256', typ: 'JWT'};
    var oPayload = {};
    oPayload.sub = window.localStorage.getItem('uid');
    oPayload.jti = window.localStorage.getItem('sid');
    oPayload.exp = KJUR.jws.IntDate.get('now + 1day');;
    var sHeader = JSON.stringify(oHeader);
    var sPayload = JSON.stringify(oPayload);
    return KJUR.jws.JWS.sign("RS512", sHeader, sPayload, priv);
  }
}

export default Account;
