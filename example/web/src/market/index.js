import './index.scss';
import './trade.scss';
import $ from 'jquery';
import uuid from 'uuid/v4';
import Chart from './chart.js';
import FormUtils from '../utils/form.js';

function Market(router, api) {
  this.router = router;
  this.api = api;
  this.templateIndex = require('./index.html');
  this.templateTrade = require('./trade.html');
  this.itemOrder = require('./order_item.html');
}

Market.prototype = {
  index: function (market) {
    if (!market) {
      var d = window.localStorage.getItem('market.default');
      if (!d || d === '') {
        d = 'BTC-USDT';
      }
      this.router.replace('/trade/' + d);
      return;
    }

    var pair = this.api.asset.market(market);
    if (!pair) {
      this.router.replace('/trade/BTC-USDT');
      return;
    }
    window.localStorage.setItem('market.default', market);
    const base = pair[0];
    const quote = pair[1];
    const self = this;

    $('body').attr('class', 'market layout');
    $('#layout-container').html(self.templateIndex({
      logoURL: require('./logo.png'),
      symbolURL: require('./symbol.png')
    })).append(self.templateTrade({
      base: base,
      quote: quote
    }));

    self.handlePageScroll(market);

    $('.layout.nav .logo a').click(function() {
      var offset = $('.layout.header').outerHeight() - $('.layout.nav').outerHeight();
      window.scrollTo({ top: offset, behavior: 'smooth' });
    });

    var offset = $('.layout.header').outerHeight() + $('.markets.container').outerHeight() - $('.layout.nav').outerHeight() + 1;
    if ($(window).scrollTop() < offset) {
      window.scrollTo({ top: offset, behavior: 'smooth' });
    }

    self.handleOrderCreate();
    self.handleFormSwitch();
    self.handleBookHistorySwitch();

    self.api.engine.subscribe(base.asset_id + '-' + quote.asset_id, function (msg) {
      self.render(msg);
    });
  },

  handleFormSwitch: function () {
    $('.type.tab').click(function () {
      var type = $(this).attr('data-type').toLowerCase();
      var side = $('.side.tab.active').attr('data-side').toLowerCase();
      $('.type.tab').removeClass('active');
      $(this).addClass('active');
      $('.trade.form form').hide();
      $('.trade.form .form.' + type + '.' + side).show();
    });
    $('.side.tab').click(function () {
      var side = $(this).attr('data-side').toLowerCase();
      var type = $('.type.tab.active').attr('data-type').toLowerCase();
      $('.side.tab').removeClass('active');
      $(this).addClass('active');
      $('.trade.form form').hide();
      $('.trade.form .form.' + type + '.' + side).show();
    });
  },

  handleBookHistorySwitch: function () {
    $('.history.tab').click(function () {
      if ($('.trade.history').width() + $('.order.book').width() < $('.orders.trades .tabs').width()) {
        return;
      }
      $('.book.tab').removeClass('active');
      $(this).addClass('active');
      $('.order.book').hide();
      $('.trade.history').show();
    });
    $('.book.tab').click(function () {
      if ($('.trade.history').width() + $('.order.book').width() < $('.orders.trades .tabs').width()) {
        return;
      }
      $('.history.tab').removeClass('active');
      $(this).addClass('active');
      $('.trade.history').hide();
      $('.order.book').show();
    });
  },

  handleOrderCreate: function () {
    const self = this;

    $('.trade.form .submit-loader').hide();
    $('.trade.form :submit').show();
    $('.trade.form :submit').prop('disabled', false);

    $('.trade.form form').submit(function (event) {
      event.preventDefault();
      var form = $(this);
      var data = FormUtils.serialize(this);
      data.trace_id = uuid().toLowerCase();
      if (data.type === 'LIMIT' && data.side === 'BID') {
        data.funds = (parseFloat(data.amount) * parseFloat(data.price)).toFixed(8);
      }
      self.api.order.create(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();
        $(':submit', form).prop('disabled', false);
        if (resp.error) {
          return;
        }
      }, data);
    });
    $('.trade.form :submit').click(function (event) {
      event.preventDefault();
      $(this).hide();
      $(this).prop('disabled', true);
      var form = $(this).parents('.trade.form form');
      $('.submit-loader', form).show();
      form.submit();
    });
  },

  handlePageScroll: function (symbol) {
    $('.market.detail.spacer').height($('.market.detail.container').outerHeight());
    $('.market.detail.container').addClass('fixed');
    $(window).scroll(function (event) {
      var scroll = $(window).scrollTop();
      var height = $('.layout.header').outerHeight();
      if (scroll - height > -128) {
        $('.layout.nav').fadeIn();
      } else if (scroll - height < -256) {
        $('.layout.nav').fadeOut();
      }

      height = $('.layout.header').outerHeight() + $('.markets.container').outerHeight();
      $('.market.detail.spacer').height($('.market.detail.container').outerHeight());
      if (scroll < $('.layout.header').outerHeight() * 2 / 3) {
        $('.markets.nav').fadeOut();
      }
      if (scroll > height - $(window).height() * 2 / 3) {
        $('.markets.nav').fadeOut();
        $('.layout.nav .title').html(symbol);
      } else {
        $('.layout.nav .title').html('USDT MARKETS');
      }
      if (scroll < height - $('.market.detail .header.container').outerHeight()) {
        $('.market.detail.container').removeClass('visible');
      } else {
        $('.market.detail.container').addClass('visible');
      }
      if (scroll < height - 4) {
        if (scroll < height - $(window).height() * 2 / 3 && $(window).width() > 1200 && scroll > $('.layout.header').outerHeight() * 2 / 3) {
          $('.markets.nav').fadeIn();
        }
        $('.market.detail.spacer').show();
        $('.market.detail.container').addClass('fixed');
      } else if (scroll > height + 4){
        $('.market.detail.spacer').hide();
        $('.market.detail.container').removeClass('fixed');
      }
      if (scroll < 256) {
        $('.market.detail.container').addClass('hidden');
      } else {
        $('.market.detail.container').removeClass('hidden');
      }
    });
  },

  fixListItemHeight: function () {
    var itemHeight = $('.order.book .ask').outerHeight();
    if (!itemHeight) {
      itemHeight = 21;
    }

    var total = $('.order.book').height() - $('.order.book .spread').outerHeight() - $('.book.tab').outerHeight();
    var count = parseInt(total / itemHeight / 2) * 2;
    var line = (total / count) + 'px';
    $('.order.book .ask').css({'line-height': line, height: line});
    $('.order.book .bid').css({'line-height': line, height: line});
    var top = -(itemHeight * $('.order.book .ask').length);
    top = top + $('.book.tab').outerHeight() + total / 2;
    $('.book.data').css({'top': top + 'px'});

    total = $('.trade.history').height() - $('.history.tab').outerHeight();
    count = parseInt(total / itemHeight);
    line = (total / count) + 'px';
    $('.trade.history .ask').css({'line-height': line, height: line});
    $('.trade.history .bid').css({'line-height': line, height: line});
  },

  renderChart: function () {
    const self = this;
    const chart = new Chart();
    if (!self.priceChart) {
      self.priceChart = chart.renderPrice($('.price.chart')[0]);
    }
    self.depthChart = chart.renderDepth($('.depth.chart')[0], self.book.bids, self.book.asks);
  },

  render: function (msg) {
    console.log(msg);
    const self = this;
    if (msg.action !== 'EMIT_EVENT') {
      return;
    }
    if (!self.book) {
      self.book = {};
    }

    var data = msg.data;
    switch (data.event) {
      case 'BOOK-T0':
        var book = data.data;
        self.book.asks = book.asks;
        self.book.bids = book.bids;
        $('.order.book .spinner-container').remove();
        $('.order.book .book.data').show();
        for (var i = 0; i < book.asks.length; i++) {
          self.orderOpenOnPage(book.asks[i]);
        }
        for (var i = 0; i < book.bids.length; i++) {
          self.orderOpenOnPage(book.bids[i]);
        }
        break;
      case 'HEARTBEAT':
        break;
      case 'ORDER-OPEN':
        self.orderOpenOnBook(data.data);
        self.orderOpenOnPage(data.data);
        break;
      case 'ORDER-CANCEL':
        self.orderRemoveFromBook(data.data);
        self.orderRemoveFromPage(data.data);
        break;
      case 'ORDER-MATCH':
        self.orderRemoveFromBook(data.data);
        self.orderRemoveFromPage(data.data);
        break;
    }

    self.renderChart();
  },

  orderOpenOnPage: function (o) {
    const self = this;
    const price = parseFloat(o.price);
    const amount = parseFloat(o.amount);

    o.sideClass = o.side.toLowerCase()
    o.price = parseFloat(o.price).toFixed(8);
    o.pricePoint = o.price.replace('.', '');
    o.amount = amount.toFixed(4);
    if ($('#order-point-' + o.pricePoint).length > 0) {
      var bo = $('#order-point-' + o.pricePoint);
      o.amount = (parseFloat(bo.attr('data-amount')) + amount).toFixed(4);
      bo.replaceWith(self.itemOrder(o));
      return;
    }

    var item = self.itemOrder(o);
    var list = $('.order.item');
    for (var i = 0; i < list.length; i++) {
      var bo = $(list[i]);
      if (price < parseFloat(bo.attr('data-price'))) {
        continue;
      }

      if (o.side !== bo.attr('data-side')) {
        $('.book.data .spread').before(item);
      } else {
        bo.before(item);
      }
      self.fixListItemHeight();
      return;
    }
    if (o.side === 'BID') {
      $('.book.data').append(item);
    } else {
      $('.book.data .spread').before(item);
    }

    self.fixListItemHeight();
  },

  orderRemoveFromPage: function (o) {
    const self = this;
    const price = parseFloat(o.price);
    const amount = parseFloat(o.amount);

    o.sideClass = o.side.toLowerCase()
    o.price = parseFloat(o.price).toFixed(8);
    o.pricePoint = o.price.replace('.', '');
    o.amount = amount.toFixed(4);
    if ($('#order-point-' + o.pricePoint).length === 0) {
      return;
    }

    var bo = $('#order-point-' + o.pricePoint);
    o.amount = parseFloat(bo.attr('data-amount')) - amount;
    if (o.amount > 0) {
      o.amount = o.amount.toFixed(4);
      bo.replaceWith(self.itemOrder(o));
    } else {
      bo.remove();
    }

    self.fixListItemHeight();
  },

  orderOpenOnBook: function (o) {
    const self = this;
    const price = parseFloat(o.price);
    const amount = parseFloat(o.amount);

    if (o.side === 'ASK') {
      for (var i = 0; i < self.book.asks.length; i++) {
        var bo = self.book.asks[i];
        if (bo.price === o.price) {
          bo.amount = parseFloat((parseFloat(bo.amount) + amount).toFixed(8));
          return;
        }
        if (parseFloat(bo.price) > price) {
          self.book.asks.splice(i, 0, o);
          return;
        }
      }
      self.book.asks.push(o);
    } else if (o.side === 'BID') {
      for (var i = 0; i < self.book.bids.length; i++) {
        var bo = self.book.bids[i];
        if (bo.price === o.price) {
          bo.amount = parseFloat((parseFloat(bo.amount) + amount).toFixed(8));
          return;
        }
        if (parseFloat(bo.price) < price) {
          self.book.bids.splice(i, 0, o);
          return;
        }
      }
      self.book.bids.push(o);
    }
  },

  orderRemoveFromBook: function (o) {
    const self = this;
    const price = parseFloat(o.price);
    const amount = parseFloat(o.amount);

    var list = self.book.asks;
    if (o.side === 'BID') {
      list = self.book.bids;
    }

    for (var i = 0; i < list.length; i++) {
      var bo = list[i];
      if (bo.price !== o.price) {
        continue;
      }

      bo.amount = parseFloat((parseFloat(bo.amount) - amount).toFixed(8));
      if (bo.amount === 0) {
        list.splice(i, 1);
      }
      return;
    }
  }
};

export default Market;
