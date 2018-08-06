import URLUtils from '../utils/url.js';

function Auth(router, api) {
  this.router = router;
  this.api = api;
}

Auth.prototype = {
  render: function () {
    const self = this;
    const error = URLUtils.getUrlParameter("error");
    const authorizationCode = URLUtils.getUrlParameter("code");
    var returnTo = URLUtils.getUrlParameter("return_to");
    if (returnTo === undefined || returnTo === null || returnTo === "") {
      returnTo = "/accounts";
    }
    returnTo = WEB_ROOT + returnTo;
    if (error === 'access_denied') {
      self.api.notify('error', i18n.t('general.errors.403'));
      window.location.replace(returnTo);
      return;
    }
    self.api.account.connectMixin(function (resp) {
      if (resp.error && resp.error.code === 403) {
        self.api.notify('error', i18n.t('general.errors.403'));
        window.location.replace(returnTo);
        return true;
      }
      if (resp.error) {
        return false;
      }
      window.location.replace(returnTo);
    }, authorizationCode);
  }
};

export default Auth;
