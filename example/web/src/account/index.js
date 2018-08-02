import './index.scss';
import $ from 'jquery';
import 'intl-tel-input/build/css/intlTelInput.css';
import 'intl-tel-input';
import FormUtils from '../utils/form.js';

function Account(router, api) {
  this.router = router;
  this.api = api;
  this.templateUser = require('./user.html');
  this.templateSession = require('./session.html');
  this.templateMe = require('./me.html');
  this.stepCode = require('./step_code.html');
  this.stepUser = require('./step_user.html');
}

Account.prototype = {
  signUp: function () {
    const self = this;
    $('body').attr('class', 'account layout');
    $('#layout-container').html(self.templateUser());

    var initialCountry = 'US';
    if (navigator.language && navigator.language.indexOf('zh') >= 0) {
      initialCountry = 'CN';
    }
    var phoneInput = $('#enroll-phone-form #phone');
    phoneInput.intlTelInput({
      "initialCountry": initialCountry
    });
    phoneInput.focus();

    $('#enroll-phone-form').submit(function (event) {
      event.preventDefault();
      var form = $(this);
      var phone = phoneInput.val().trim();
      if (phone.indexOf('+') !== 0) {
        phone = '+' + phoneInput.intlTelInput("getSelectedCountryData")['dialCode'] + phone;
      }
      var params = {
        category: 'PHONE',
        receiver: phone
      };
      self.api.account.newVerification(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();

        if (resp.error) {
          return;
        }
        self.api.notify('success', i18n.t('account.notifications.phone.verification.send.success'));
        self.renderCodeStep(phone, resp.data.verification_id, 'USER');
      }, params);
    });
    $('#enroll-phone-form :submit').click(function (event) {
      event.preventDefault();
      var form = $(this).parents('form');
      $('.submit-loader', form).show();
      $(this).hide();
      form.submit();
    });
  },

  renderCodeStep: function (phone, verificationId, purpose) {
    const self = this;
    $('body').attr('class', 'account layout');
    $('#layout-container').html(self.stepCode({
      phone: phone,
      verificationId: verificationId
    }));
    $('#enroll-verify-form #code').focus();

    $('#enroll-verify-form').submit(function (event) {
      event.preventDefault();
      var form = $(this);
      var params = FormUtils.serialize(form);
      self.api.account.verifyVerification(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();

        if (resp.error) {
          return;
        }
        switch (purpose) {
          case 'USER':
            self.renderUserStep(verificationId);
            break;
        }
      }, params);
    });
    $('#enroll-verify-form :submit').click(function (event) {
      event.preventDefault();
      var form = $(this).parents('form');
      $('.submit-loader', form).show();
      $(this).hide();
      form.submit();
    });
  },

  renderUserStep: function (verificationId) {
    const self = this;
    $('body').attr('class', 'account layout');
    $('#layout-container').html(self.stepUser({
      verificationId: verificationId
    }));
    $('#enroll-verify-form #password').focus();

    $('#enroll-verify-form').submit(function (event) {
      event.preventDefault();
      var form = $(this);
      var params = FormUtils.serialize(form);
      self.api.account.createUser(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();

        if (resp.error) {
          return;
        }
        self.router.replace('/me');
      }, params);
    });
    $('#enroll-verify-form :submit').click(function (event) {
      event.preventDefault();
      var form = $(this).parents('form');
      $('.submit-loader', form).show();
      $(this).hide();
      form.submit();
    });
  },

  signIn: function () {
    const self = this;
    $('body').attr('class', 'account layout');
    $('#layout-container').html(self.templateSession());
  },

  me: function () {
    const self = this;
    $('body').attr('class', 'account layout');
    $('#layout-container').html(self.templateMe());
    self.api.account.me(function (resp) {
      if (resp.error) {
        return;
      }
    });
    self.api.mixin.assets(function (resp) {
      console.log(resp);
    });
    self.api.ocean.orders(function (resp) {
      console.log(resp);
    });
  }
};

export default Account;
