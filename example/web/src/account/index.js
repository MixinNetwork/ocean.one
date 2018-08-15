import './index.scss';
import $ from 'jquery';
import 'intl-tel-input/build/css/intlTelInput.css';
import 'intl-tel-input';
import uuid from 'uuid/v4';
import QRious from 'qrious';
import FormUtils from '../utils/form.js';
import TimeUtils from '../utils/time.js';
import URLUtils from '../utils/url.js';

function Account(router, api) {
  this.router = router;
  this.api = api;
  this.templateUser = require('./user.html');
  this.templateSession = require('./session.html');
  this.templateOrders = require('./orders.html');
  this.templateAssets = require('./assets.html');
  this.templateAsset = require('./asset.html');
  this.stepCode = require('./step_code.html');
  this.stepUser = require('./step_user.html');
  this.stepPassword = require('./step_password.html');
}

Account.prototype = {
  signUp: function () {
    if (this.api.account.token() !== '') {
      this.router.replace('/accounts');
      return;
    }

    this.sendVerification('USER');
  },

  resetPassword: function () {
    if (this.api.account.token() !== '') {
      this.router.replace('/accounts');
      return;
    }

    this.sendVerification('PASSWORD');
  },

  sendVerification: function (purpose) {
    const self = this;
    $('body').attr('class', 'account layout');
    var title = 'home.sign.up';
    if (purpose === 'PASSWORD') {
      title = 'account.buttons.reset.password';
    }
    $('#layout-container').html(self.templateUser({title: window.i18n.t(title)}));

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
        receiver: phone,
        recaptcha_response: $('#recaptcha-response').val()
      };
      self.api.account.newVerification(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();

        if (resp.error) {
          return;
        }
        self.api.notify('success', i18n.t('account.notifications.phone.verification.send.success'));
        self.renderCodeStep(phone, resp.data.verification_id, purpose);
      }, params);
    });
    var enroll = function (token) {
      $('#recaptcha-response').val(token);
      $('#enroll-phone-form').submit();
    };
    $('#enroll-phone-form :submit').click(function (event) {
      event.preventDefault();
      var form = $(this).parents('form');
      $('.submit-loader', form).show();
      $(this).hide();

      var widgetId = grecaptcha.render("g-recaptcha", {
        "sitekey": RECAPTCHA_SITE_KEY,
        "size": "invisible",
        "callback": enroll
      });
      grecaptcha.execute(widgetId);
    });
  },

  renderCodeStep: function (phone, verificationId, purpose) {
    const self = this;
    var title = 'home.sign.up';
    if (purpose === 'PASSWORD') {
      title = 'account.buttons.reset.password';
    }
    $('body').attr('class', 'account layout');
    $('#layout-container').html(self.stepCode({
      title: window.i18n.t(title),
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
          case 'PASSWORD':
            self.renderResetPassword(verificationId);
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
        self.router.replace('/accounts');
      }, params);
    });
    $('#enroll-verify-form :submit').click(function (event) {
      event.preventDefault();
      if ($('#password').val() !== $('#password-confirmation').val()) {
        self.api.notify('error', i18n.t('account.notifications.password.mismatch'));
        return;
      }
      var form = $(this).parents('form');
      $('.submit-loader', form).show();
      $(this).hide();
      form.submit();
    });
  },

  renderResetPassword: function (verificationId) {
    const self = this;
    $('body').attr('class', 'account layout');
    $('#layout-container').html(self.stepPassword({
      verificationId: verificationId
    }));
    $('#enroll-verify-form #password').focus();

    $('#enroll-verify-form').submit(function (event) {
      event.preventDefault();
      var form = $(this);
      var params = FormUtils.serialize(form);
      self.api.account.resetPassword(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();

        if (resp.error) {
          return;
        }
        self.router.replace('/accounts');
      }, params);
    });
    $('#enroll-verify-form :submit').click(function (event) {
      event.preventDefault();
      if ($('#password').val() !== $('#password-confirmation').val()) {
        self.api.notify('error', i18n.t('account.notifications.password.mismatch'));
        return;
      }
      var form = $(this).parents('form');
      $('.submit-loader', form).show();
      $(this).hide();
      form.submit();
    });
  },

  signIn: function () {
    if (this.api.account.token() !== '') {
      this.router.replace('/accounts');
      return;
    }

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
        self.router.replace('/accounts');
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
        preset[i].price_usd = '0';
        preset[i].balance = '0';
        preset[i].depositEnabled = true;
        resp.data.push(preset[i]);
      }
      resp.data.sort(function (a, b) {
        var at = parseFloat(a.price_usd) * parseFloat(a.balance);
        var bt = parseFloat(b.price_usd) * parseFloat(b.balance);
        if (at > bt) {
          return -1;
        }
        if (at < bt) {
          return 1;
        }
        if (a.symbol < b.symbol) {
          return -1;
        }
        if (a.symbol > b.symbol) {
          return 1;
        }
        return 0;
      });
      $('body').attr('class', 'account layout');
      $('#layout-container').html(self.templateAssets({
        assets: resp.data,
      }));
      self.router.updatePageLinks();
    });
  },

  assetDo: function (id, action, me) {
    const self = this;
    self.api.mixin.asset(function (resp) {
      if (resp.error) {
        return;
      }
      resp.data.me = me;
      resp.data.trace_id = uuid().toLowerCase();
      $('body').attr('class', 'account layout');
      $('#layout-container').html(self.templateAsset(resp.data));
      self.router.updatePageLinks();
      $('.tab').removeClass('active');
      $('.tab.' + action.toLowerCase()).addClass('active');
      $('.action.container').hide();
      $('.action.container.' + action.toLowerCase()).show();

      if (action === 'WITHDRAWAL') {
        return self.handleWithdrawal(me, resp.data);
      }

      if (resp.data.public_key !== '') {
        $('.address.deposit.container').show();
        new QRious({
          element: $('.deposit.address.code.container canvas')[0],
          value: resp.data.public_key,
          size: 500
        });
      } else if (resp.data.account_name !== '') {
        $('.account.deposit.container').show();
        new QRious({
          element: $('.deposit.account.name.code.container canvas')[0],
          value: resp.data.account_name,
          size: 500
        });
        new QRious({
          element: $('.deposit.account.tag.code.container canvas')[0],
          value: resp.data.account_tag,
          size: 500
        });
      }
    }, id);
  },

  handleWithdrawal: function (me, asset) {
    const self = this;
    if (me.mixin_id && me.mixin_id !== '') {
      $('.mixin.connected').show();
      $('.mixin.disconnected').hide();
    } else {
      $('.mixin.connected').hide();
      $('.mixin.disconnected').show();
    }
    $('.withdrawal.form').submit(function (event) {
      event.preventDefault();
      var form = $(this);
      var params = FormUtils.serialize(form);
      self.api.withdrawal.create(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();

        if (resp.error) {
          return;
        }
        self.api.router.replace('/accounts');
      }, params);
    });
    $('.withdrawal.form :submit').click(function (event) {
      event.preventDefault();
      var form = $(this).parents('form');
      $('.submit-loader', form).show();
      $(this).hide();
      form.submit();
    });
  },

  asset: function (id, action) {
    const self = this;
    if (action === "DEPOSIT") {
      return self.assetDo(id, action);
    }
    self.api.account.me(function (resp) {
      if (resp.error) {
        return;
      }
      return self.assetDo(id, action, resp.data);
    });
  },

  orders: function (market) {
    var pair = this.api.asset.market(market);
    if (!pair) {
      this.router.replace('/orders/BTC-USDT');
      return;
    }
    var state = "PENDING";
    if (URLUtils.getUrlParameter('state') === "DONE") {
      state = "DONE"
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
        pending: state === "PENDING",
        base: base,
        quote: quote,
        pair: base.symbol+'-'+quote.symbol,
        orders: resp.data
      }));
      self.handleOrderCancel();
      self.router.updatePageLinks();
    }, state, 'DESC', base.asset_id + '-' + quote.asset_id, offset);
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
