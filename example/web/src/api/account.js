import KJUR from 'jsrsasign';

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

  check: function (callback) {
    const self = this;
    this.api.request('GET', '/me', undefined, function(resp) {
      if (typeof callback === 'function') {
        return callback(resp);
      }
    });
  },

  token: function () {
    var priv = window.localStorage.getItem('token');
    if (!priv) {
      return "";
    }
    var oHeader = {alg: 'RS512', typ: 'JWT'};
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
