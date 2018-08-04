import './index.scss';
import $ from 'jquery';
import 'intl-tel-input/build/css/intlTelInput.css';
import 'intl-tel-input';
import FormUtils from '../utils/form.js';
import TimeUtils from '../utils/time.js';

function Account(router, api) {
  this.router = router;
  this.api = api;
  this.templateUser = require('./user.html');
  this.templateSession = require('./session.html');
  this.templateMe = require('./me.html');
  this.templateOrders = require('./orders.html');
  this.templateAssets = require('./assets.html');
  this.templateAsset = require('./asset.html');
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
      var params = FormUtils.serialize(form);
      params.phone = phone;
      self.api.account.createSession(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();

        if (resp.error) {
          return;
        }
        self.router.replace('/me');
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

  me: function () {
    const self = this;
    $('body').attr('class', 'account layout');
    $('#layout-container').html(self.templateMe());
    self.api.account.me(function (resp) {
      if (resp.error) {
        return;
      }
    });
  },

  assets: function () {
    const self = this;
    const preset = self.api.asset.preset();
    self.api.mixin.assets(function (resp) {
      if (resp.error) {
        return;
      }
      var filter = {};
      for (var i = 0; i < resp.data.length; i++) {
        var a = resp.data[i];
        filter[a.asset_id] = true;
        a.depositEnabled = a.asset_id != 'de5a6414-c181-3ecc-b401-ce375d08c399';
      }
      for (var i = 0; i < preset.length; i++) {
        if (filter[preset[i].asset_id]) {
          continue;
        }
        preset[i].balance = '0';
        preset[i].depositEnabled = true;
        resp.data.push(preset[i]);
      }
      $('body').attr('class', 'account layout');
      $('#layout-container').html(self.templateAssets({
        assets: resp.data,
      }));
      self.router.updatePageLinks();
    });
  },

  asset: function (id, action) {
    const self = this;
    self.api.mixin.asset(function (resp) {
      if (resp.error) {
        return;
      }
      $('body').attr('class', 'account layout');
      $('#layout-container').html(self.templateAsset(resp.data));
      self.router.updatePageLinks();
      $('.tab').removeClass('active');
      $('.tab.' + action.toLowerCase()).addClass('active');
      $('.action.container').hide();
      $('.action.container.' + action.toLowerCase()).show();
    }, id);
  },

  orders: function (market) {
    var pair = this.api.asset.market(market);
    if (!pair) {
      this.router.replace('/orders/BTC-USDT');
      return;
    }
    const base = pair[0];
    const quote = pair[1];

    const self = this;
    var offset = TimeUtils.rfc3339(new Date());
    self.api.ocean.orders(function (resp) {
      if (resp.error) {
        return;
      }
      for (var i = 0; i < resp.data.length; i++) {
        var o = resp.data[i];
        o.created_at = TimeUtils.short(o.created_at);
        o.amount = parseFloat((parseFloat(o.filled_amount) + parseFloat(o.remaining_amount)).toFixed(8));
        o.funds = parseFloat((parseFloat(o.filled_funds) + parseFloat(o.remaining_funds)).toFixed(8));
        if (o.side === 'BID') {
          o.amount = parseFloat((o.funds / parseFloat(o.price)).toFixed(8));
        }
        o.filled_price = 0;
        if (o.filled_amount !== '0') {
          o.filled_price = parseFloat((parseFloat(o.filled_funds) / parseFloat(o.filled_amount)).toFixed(8));
        }
      }
      $('body').attr('class', 'account layout');
      $('#layout-container').html(self.templateOrders({
        base: base,
        quote: quote,
        orders: resp.data
      }));
      self.handleOrderCancel();
      self.router.updatePageLinks();
    }, base.asset_id + '-' + quote.asset_id, offset);
  },

  handleOrderCancel: function () {
    const self = this;
    $('.orders.list .cancel.action a').click(function () {
      var item = $(this).parents('.order.item');
      var id = $(item).attr('data-id');
      self.api.order.cancel(function (resp) {
        if (resp.error) {
          return;
        }
        $(item).fadeOut().remove();
      }, id);
    });
  }
};

export default Account;
