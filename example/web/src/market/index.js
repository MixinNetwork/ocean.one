import './index.scss';
import './trade.scss';
import $ from 'jquery';
import jQueryColor from '../jquery-color-plus-names.js';
import uuid from 'uuid/v4';
import {BigNumber} from 'bignumber.js';
import Chart from './chart.js';
import FormUtils from '../utils/form.js';
import TimeUtils from '../utils/time.js';

function Market(router, api) {
  this.router = router;
  this.api = api;
  this.templateIndex = require('./index.html');
  this.templateTrade = require('./trade.html');
  this.itemOrder = require('./order_item.html');
  this.itemTrade = require('./trade_item.html');
  this.itemMarket = require('./market_item.html');
  jQueryColor($);
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

    const self = this;
    self.base = pair[0];
    self.quote = pair[1];
    self.api.market.index(function (resp) {
      if (resp.error) {
        return;
      }
      self.do(resp.data);
    });
  },

  do: function (inputs) {
    const self = this;

    if (self.quote.asset_id === '815b0b1a-2764-3736-8faa-42d694fa620a') {
      self.quote.step = '0.0001';
    } else {
      self.quote.step = '0.00000001';
    }

    $('body').attr('class', 'market layout');
    $('#layout-container').html(self.templateIndex({
      logoURL: require('./logo.png'),
      symbolURL: require('./symbol.png'),
      title: self.base.symbol + '-' + self.quote.symbol
    })).append(self.templateTrade({
      base: self.base,
      quote: self.quote,
      trace: uuid().toLowerCase()
    }));

    if (self.api.account.token() === '') {
      $('.account.sign.out.button').hide();
      $('.account.sign.in.button').show();
      $('.account.in.actions').hide();
      $('.account.out.actions').show();
    } else {
      $('.account.sign.in.button').hide();
      $('.account.sign.out.button').show();
      $('.account.in.actions').show();
      $('.account.out.actions').hide();
    }

    $('.markets.container').on('click', '.market.item', function (event) {
      event.preventDefault();
      if ($(this).data('symbol') === self.base.symbol + '-' + self.quote.symbol) {
        $('.layout.header').toggleClass('invisible');
        $('.market.detail.container').slideToggle();
        $('.markets.container').toggle();
        $('.layout.header').toggle();
        $('.layout.nav').show();
      } else {
        window.location.href = '/trade/' + $(this).data('symbol');
      }
    });
    var markets = self.renderMarkets(inputs);
    setInterval(function() {
      self.pollMarkets();
    }, 5000);

    for (var i = 0; i < markets.length; i++) {
      var m = markets[i];
      if (m.base.asset_id === self.base.asset_id && m.quote.asset_id === self.quote.asset_id) {
        self.updateTickerPrice(m.price);
      }
    }

    self.handlePageScroll();

    $('.layout.nav .logo a, .layout.nav .title').click(function(event) {
      event.preventDefault();
      $('.layout.header').toggleClass('invisible');
      if ($('.layout.header').hasClass('invisible')) {
        $('.market.detail.container').slideToggle();
        $('.markets.container').toggle();
        $('.layout.header').toggle();
        $('.layout.nav').show();
      } else {
        $('.market.detail.container').slideToggle();
        $('.markets.container').toggle();
        $('.layout.header').slideToggle();
        window.scrollTo({top: $('.layout.header').outerHeight() - $('.layout.nav').outerHeight(), behavior: 'instant'});
      }
    });

    $('.order.book').on('click', 'li', function (event) {
      event.preventDefault();
      $('.trade.form input[name="price"]').val($(this).data('price'));
    });

    self.handleOrderCreate();
    self.handleFormSwitch();
    self.handleBookHistorySwitch();
    self.fixListItemHeight();

    var pollBalance = function () {
      self.pollAccountBalance(self.base.asset_id);
      self.pollAccountBalance(self.quote.asset_id);
    };
    pollBalance();
    setInterval(pollBalance, 7000);

    var fetchTrades = function () {
      var offset = TimeUtils.rfc3339(new Date());
      self.api.ocean.trades(function (resp) {
        if (resp.error) {
          return true;
        }
        var trades = resp.data;
        for (var i = trades.length; i > 0; i--) {
          self.addTradeEntry(trades[i-1]);
        }
        $('.trade.history .spinner-container').remove();
        self.fixListItemHeight();
      }, self.base.asset_id + '-' + self.quote.asset_id, offset);
    };
    setTimeout(function() { fetchTrades(); }, 500);

    self.pollCandles(300);
    self.candleInterval = setInterval(function () {
      self.pollCandles(300);
    }, 60000);
    self.handleCandleSwitch();

    self.api.engine.subscribe(self.base.asset_id + '-' + self.quote.asset_id, function (msg) {
      self.render(msg);
    });
  },

  renderMarkets: function (inputs) {
    const self = this;
    var markets = [];
    for (var i = 0; i < inputs.length; i++) {
      var m = inputs[i];
      m.base = self.api.asset.getById(m.base);
      m.quote = self.api.asset.getById(m.quote);
      if (m.base && m.quote) {
        markets.push(m);
      }
    }

    markets.sort(function (a, b) {
      if (a.quote.symbol < b.quote.symbol) {
        return -1;
      }
      if (a.quote.symbol > b.quote.symbol) {
        return 1;
      }
      var at = new BigNumber(a.total);
      var bt = new BigNumber(b.total);
      if (at.isGreaterThan(bt)) {
        return -1;
      }
      if (at.isLessThan(bt)) {
        return 1;
      }
      if (a.base.symbol < b.base.symbol) {
        return -1;
      }
      if (a.base.symbol > b.base.symbol) {
        return 1;
      }
      return 0;
    });

    for (var i = 0; i < markets.length; i++) {
      var m = markets[i];
      m.change_amount = new BigNumber(m.price).minus(new BigNumber(m.price).div(new BigNumber(m.change).plus(1))).toFixed(8).replace(/\.?0+$/,"");
      if (m.quote.asset_id === '815b0b1a-2764-3736-8faa-42d694fa620a') {
        m.change_amount = new BigNumber(m.change_amount).toFixed(4).replace(/\.?0+$/,"");
      }
      m.direction = m.change < 0 ? 'down' : 'up';
      m.change = (m.change < 0 ? '' : '+') + Number(m.change * 100).toFixed(2) + '%';
      m.volume = new BigNumber(m.volume).toFixed(2);
      m.total = new BigNumber(m.total).toFixed(2);
      m.price_usd = new BigNumber(m.price).times(m.quote_usd);
      if (m.price_usd.toFixed(4).indexOf('0.00') === 0) {
        m.price_usd = new BigNumber(m.price_usd).toFixed(4);
      } else {
        m.price_usd = new BigNumber(m.price_usd).toFixed(2);
      }
      if (self.base.asset_id === m.base.asset_id && self.quote.asset_id === m.quote.asset_id) {
        self.quote_usd = m.quote_usd;
        $('.ticker.change').removeClass('up');
        $('.ticker.change').removeClass('down');
        $('.ticker.change').addClass(m.direction);
        $('.ticker.change .value').html(m.change);
        $('.ticker.volume .value').html(m.volume);
        $('.ticker.total .value').html(m.total);
      }

      m.price = m.price.toFixed(8).replace(/\.?0+$/,"");
      var item = $('#market-item-' + m.base.symbol + '-' + m.quote.symbol);
      if (item.length > 0) {
        item.replaceWith(self.itemMarket(m));
      } else {
        $('.' + m.quote.symbol.toLowerCase() + '.markets.block table tbody').append(self.itemMarket(m));
      }
      var cell = $('#market-item-' + m.base.symbol + '-' + m.quote.symbol + ' .change.cell');
      cell.removeClass('up');
      cell.removeClass('down');
      cell.addClass(m.direction);
      cell = $('#market-item-' + m.base.symbol + '-' + m.quote.symbol + ' .price.cell');
      cell.removeClass('up');
      cell.removeClass('down');
      cell.addClass(m.direction);
    }

    var totalUSD = 0;
    for (var i = 0; i < markets.length; i++) {
      var m = markets[i];
      totalUSD += m.total * m.quote_usd;
    }
    console.log("24 hour volume in USD " + totalUSD.toLocaleString(undefined, { maximumFractionDigits: 0 }));

    return markets;
  },

  pollMarkets: function () {
    const self = this;
    self.api.market.index(function (resp) {
      if (resp.error) {
        return true;
      }

      self.renderMarkets(resp.data);
    });
  },

  handleFormSwitch: function () {
    $('.type.tab').click(function (event) {
      event.preventDefault();
      var type = $(this).attr('data-type').toLowerCase();
      var side = $('.side.tab.active').attr('data-side').toLowerCase();
      $('.type.tab').removeClass('active');
      $(this).addClass('active');
      $('.trade.form form').hide();
      $('.trade.form .form.' + type + '.' + side).show();
    });
    $('.side.tab').click(function (event) {
      event.preventDefault();
      var side = $(this).attr('data-side').toLowerCase();
      var type = $('.type.tab.active').attr('data-type').toLowerCase();
      $('.side.tab').removeClass('active');
      $(this).addClass('active');
      $('.trade.form form').hide();
      $('.trade.form .form.' + type + '.' + side).show();
    });
  },

  handleBookHistorySwitch: function () {
    $('.history.tab').click(function (event) {
      event.preventDefault();
      if ($('.trade.history').width() + $('.order.book').width() < $('.orders.trades .tabs').width()) {
        return;
      }
      $('.book.tab').removeClass('active');
      $(this).addClass('active');
      $('.order.book').hide();
      $('.trade.history').show();
    });
    $('.book.tab').click(function (event) {
      event.preventDefault();
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

    $('.trade.form form').submit(function (event) {
      event.preventDefault();
      var form = $(this);
      var data = FormUtils.serialize(this);
      if (data.type === 'LIMIT' && data.side === 'BID') {
        data.funds = new BigNumber(data.amount).times(data.price).toFixed(8);
      }
      self.api.order.create(function (resp) {
        $('.submit-loader', form).hide();
        $(':submit', form).show();
        $(':submit', form).prop('disabled', false);
        if (resp.error) {
          return;
        }

        $('.trade.form input[name="amount"]').val('');
        $('.trade.form input[name="funds"]').val('');
        $('.trade.form input[name="trace_id"]').val(uuid().toLowerCase());
        if (data.side === 'BID') {
          self.pollAccountBalance($('.trade.form form input[name="quote"]').val());
        } else {
          self.pollAccountBalance($('.trade.form form input[name="base"]').val());
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

  handlePageScroll: function () {
    $(window).scroll(function (event) {
      if (!$('.markets.container').is(':visible')) {
        return;
      }

      var scroll = $(window).scrollTop();
      var height = $('.layout.header').outerHeight();
      if (scroll - height > -128) {
        $('.layout.nav').fadeIn();
      } else if (scroll - height < -256) {
        $('.layout.nav').fadeOut();
      }
    });
  },

  fixListItemHeight: function () {
    var mass = $('.book.data .ask').length - 60;
    if (mass > 0) {
      $('.book.data li.ask:nth-of-type(-1n+' + mass + ')').remove();
    }
    mass = $('.book.data li.ask').length + 60;
    $('.book.data li.bid:nth-of-type(1n+' + mass + ')').remove();

    const itemHeight = 21;
    var total = $('.order.book').height() - $('.order.book .spread').outerHeight() - $('.book.tab').outerHeight();
    var count = parseInt(total / itemHeight / 2) * 2;
    var line = (total / count) + 'px';
    $('.order.book .ask').css({'line-height': line, height: line});
    $('.order.book .bid').css({'line-height': line, height: line});
    var top = -(total / count * $('.order.book .ask').length);
    top = top + $('.book.tab').outerHeight() + total / 2;
    $('.book.data').css({'top': top + 'px'});

    total = $('.trade.history').height() - $('.history.tab').outerHeight();
    count = parseInt(total / itemHeight);
    line = (total / count) + 'px';
    $('.trade.history .ask').css({'line-height': line, height: line});
    $('.trade.history .bid').css({'line-height': line, height: line});
  },

  handleCandleSwitch: function () {
    const self = this;
    $('.charts.container .tabs li').click(function (event) {
      event.preventDefault();
      $('.charts.container .tabs li').removeClass('active');
      $(this).addClass('active');
      if ($(this).hasClass('depth')) {
        $('.price.chart').hide();
        $('.depth.chart').show();
        return;
      }

      if (($('.price.chart').outerHeight() * 3 / 2) > $('.charts.container').outerHeight()) {
        $('.depth.chart').hide();
      }
      $('.price.chart').show();
      const granularity = $(this).data('granularity');
      clearInterval(self.candleInterval);
      self.pollCandles(granularity);
      self.candleInterval = setInterval(function () {
        self.pollCandles(granularity);
      }, 60000);
    });
  },

  pollCandles: function (granularity) {
    const self = this;
    self.api.market.candles(function (resp) {
      if (resp.error) {
        return true;
      }
      self.renderCandleChart(resp.data);
    }, self.base.asset_id + '-' + self.quote.asset_id, granularity);
  },

  renderCandleChart: function (data) {
    const self = this;
    const chart = new Chart();
    if (!self.priceChart) {
      self.priceChart = chart.renderPrice($('.price.chart')[0], self.base.symbol, data);
    } else {
      data = chart.prepareCandleData(data);
      var ohlc = data[0];
      var volume = data[1];
      self.priceChart.series[0].setData(volume);
      self.priceChart.series[1].setData(ohlc);
    }
  },

  renderDepthChart: function () {
    const self = this;
    const chart = new Chart();
    self.depthChart = chart.renderDepth($('.depth.chart')[0], self.book.bids, self.book.asks);
  },

  render: function (msg) {
    const self = this;
    if (msg.action !== 'EMIT_EVENT') {
      return;
    }
    if (!self.book) {
      self.book = {
        asks: [],
        bids: []
      };
    }

    var data = msg.data;
    switch (data.event) {
      case 'BOOK-T0':
        var book = data.data;
        self.book.asks = book.asks;
        self.book.bids = book.bids;
        $('.order.book .spinner-container').remove();
        $('.order.book .book.data').show();
        $('.order.book .order.item').remove();
        for (var i = 0; i < book.asks.length; i++) {
          self.orderOpenOnPage(book.asks[i], true);
        }
        for (var i = 0; i < book.bids.length; i++) {
          self.orderOpenOnPage(book.bids[i], true);
        }
        self.fixListItemHeight();
        break;
      case 'HEARTBEAT':
        return;
      case 'ORDER-OPEN':
        $('.order.book .spinner-container').remove();
        $('.order.book .book.data').show();
        self.orderOpenOnBook(data.data);
        self.orderOpenOnPage(data.data);
        self.fixListItemHeight();
        break;
      case 'ORDER-CANCEL':
        self.orderRemoveFromBook(data.data);
        self.orderRemoveFromPage(data.data);
        self.fixListItemHeight();
        break;
      case 'ORDER-MATCH':
        data.data.created_at = data.timestamp;
        self.updateTickerPrice(data.data.price);
        self.addTradeEntry(data.data);
        self.orderRemoveFromBook(data.data);
        self.orderRemoveFromPage(data.data);
        self.fixListItemHeight();
        break;
    }

    self.renderDepthChart();
  },

  updateTickerPrice: function (price) {
    const self = this;
    $('.book.data .spread').attr('data-price', price);
    $('.quote.price').html(new BigNumber(price).toFixed(8).replace(/\.?0+$/,""));
    var price_usd = new BigNumber(price).times(self.quote_usd);
    if (price_usd.toFixed(4).indexOf('0.00') === 0) {
      price_usd = new BigNumber(price_usd).toFixed(4);
    } else {
      price_usd = new BigNumber(price_usd).toFixed(2);
    }
    $('.fiat.price').html('$' + price_usd);
  },

  addTradeEntry: function (o) {
    const self = this;
    if ($('#trade-item-' + o.trade_id).length > 0) {
      return;
    }
    var items = $('.trade.item');
    if (items.length > 0 && new Date($(items[0]).attr('data-time')) > new Date(o.created_at)) {
      return;
    }
    $('.trade.history .spinner-container').remove();
    if (self.quote.asset_id === '815b0b1a-2764-3736-8faa-42d694fa620a') {
      o.price = new BigNumber(o.price).toFixed(4);
    } else {
      o.price = new BigNumber(o.price).toFixed(8);
    }
    o.amount = new BigNumber(o.amount).toFixed(4);
    if (o.amount === '0.0000') {
      o.amount = '0.0001';
    }
    o.sideClass = o.side.toLowerCase();
    o.time = TimeUtils.short(o.created_at);
    $('.history.data').prepend(self.itemTrade(o));
    $('.history.data li:nth-of-type(1n+100)').remove();
  },

  orderOpenOnPage: function (o, instant) {
    const self = this;
    const price = new BigNumber(o.price);
    const amount = new BigNumber(o.amount);
    var bgColor = 'rgba(0, 181, 110, 0.3)';
    if (o.side === 'ASK') {
      bgColor = 'rgba(229, 85, 65, 0.3)';
    }

    o.sideClass = o.side.toLowerCase()
    if (self.quote.asset_id === '815b0b1a-2764-3736-8faa-42d694fa620a') {
      o.price = new BigNumber(o.price).toFixed(4);
    } else {
      o.price = new BigNumber(o.price).toFixed(8);
    }
    o.pricePoint = o.price.replace('.', '');
    o.amount = amount.toFixed(4);;
    if (o.amount === '0.0000') {
      o.amount = '0.0001';
    }
    if ($('#order-point-' + o.pricePoint).length > 0) {
      var bo = $('#order-point-' + o.pricePoint);
      o.amount = new BigNumber(bo.attr('data-amount')).plus(amount).toFixed(4);
      if (instant) {
        bo.replaceWith(self.itemOrder(o));
      } else {
        bo.replaceWith($(self.itemOrder(o)).css('background-color', bgColor).animate({ backgroundColor: "transparent" }, 500));
      }
      return;
    }

    var list = $('.order.item');
    var item = self.itemOrder(o);
    if (!instant) {
      item = $(item).css('background-color', bgColor).animate({ backgroundColor: "transparent" }, 500);
    }
    for (var i = 0; i < list.length; i++) {
      var bo = $(list[i]);
      if (price.isLessThan(bo.attr('data-price'))) {
        continue;
      }

      if (o.side !== bo.attr('data-side')) {
        $('.book.data .spread').before(item);
      } else {
        bo.before(item);
      }
      return;
    }
    if (o.side === 'BID') {
      $('.book.data').append(item);
    } else {
      $('.book.data .spread').before(item);
    }
  },

  orderRemoveFromPage: function (o) {
    const self = this;
    const price = new BigNumber(o.price);
    const amount = new BigNumber(o.amount);

    o.sideClass = o.side.toLowerCase()
    if (self.quote.asset_id === '815b0b1a-2764-3736-8faa-42d694fa620a') {
      o.price = new BigNumber(o.price).toFixed(4);
    } else {
      o.price = new BigNumber(o.price).toFixed(8);
    }
    o.pricePoint = o.price.replace('.', '');
    if ($('#order-point-' + o.pricePoint).length === 0) {
      return;
    }

    var bo = $('#order-point-' + o.pricePoint);
    var bgColor = 'rgba(0, 181, 110, 0.3)';
    if (o.side === 'ASK') {
      bgColor = 'rgba(229, 85, 65, 0.3)';
    }
    o.amount = new BigNumber(bo.attr('data-amount')).minus(amount);
    o.funds = new BigNumber(bo.attr('data-funds')).minus(o.funds);
    if (o.amount.isLessThan(0) || o.funds.isLessThan(0)) {
      bo.remove();
    } else {
      o.amount = o.amount.toFixed(4);
      if (o.amount === '0.0000') {
        o.amount = '0.0001';
      }
      bo.replaceWith($(self.itemOrder(o)).css('background-color', bgColor).animate({ backgroundColor: "transparent" }, 500));
    }
  },

  orderOpenOnBook: function (o) {
    const self = this;
    const price = new BigNumber(o.price);
    const amount = new BigNumber(o.amount);

    if (o.side === 'ASK') {
      for (var i = 0; i < self.book.asks.length; i++) {
        var bo = self.book.asks[i];
        var bp = new BigNumber(bo.price);
        if (bp.isEqualTo(price)) {
          bo.amount = new BigNumber(bo.amount).plus(amount).toFixed(4);
          return;
        }
        if (bp.isGreaterThan(price)) {
          self.book.asks.splice(i, 0, o);
          return;
        }
      }
      self.book.asks.push(o);
    } else if (o.side === 'BID') {
      for (var i = 0; i < self.book.bids.length; i++) {
        var bo = self.book.bids[i];
        var bp = new BigNumber(bo.price);
        if (bp.isEqualTo(price)) {
          bo.amount = new BigNumber(bo.amount).plus(amount).toFixed(4);
          return;
        }
        if (bp.isLessThan(price)) {
          self.book.bids.splice(i, 0, o);
          return;
        }
      }
      self.book.bids.push(o);
    }
  },

  orderRemoveFromBook: function (o) {
    const self = this;
    const price = new BigNumber(o.price);
    const amount = new BigNumber(o.amount);

    var list = self.book.asks;
    if (o.side === 'BID') {
      list = self.book.bids;
    }

    for (var i = 0; i < list.length; i++) {
      var bo = list[i];
      if (!new BigNumber(bo.price).isEqualTo(price)) {
        continue;
      }

      bo.amount = new BigNumber(bo.amount).minus(amount).toFixed(4);
      if (bo.amount === '0.0000') {
        list.splice(i, 1);
      }
      return;
    }
  },

  pollAccountBalance: function (asset) {
    if (this.api.account.token() === '') {
      $('.account.balances .balance').hide();
      $('.account.in.actions').hide();
      $('.account.out.actions').show();
      return;
    }
    $('.account.in.actions').show();
    $('.account.out.actions').hide();

    const self = this;
    self.api.mixin.asset(function (resp) {
      if (resp.error) {
        if (resp.error.code === 401) {
          self.api.account.clear();
        }
        return true;
      }

      var data = resp.data;
      $('.balance.' + data.symbol).css({display: 'flex'});
      $('.asset.amount.' + data.symbol).html(data.balance);
    }, asset);
  }
};

export default Market;
