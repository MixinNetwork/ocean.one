import './index.scss';
import './trade.scss';
import $ from 'jquery';
import uuid from 'uuid/v4';
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

    self.base = base;
    self.quote = quote;
    self.api.ocean.ticker(function (resp) {
      if (resp.error) {
        return;
      }
      var ticker = resp.data;

      var offset = TimeUtils.rfc3339(new Date());
      self.api.ocean.trades(function (resp) {
        if (resp.error) {
          return;
        }
        var trades = resp.data;

        $('body').attr('class', 'market layout');
        $('#layout-container').html(self.templateIndex({
          logoURL: require('./logo.png'),
          symbolURL: require('./symbol.png')
        })).append(self.templateTrade({
          base: base,
          quote: quote,
          trace: uuid().toLowerCase()
        }));

        $('.quote.price').html(ticker.price);
        for (var i = trades.length; i > 0; i--) {
          self.addTradeEntry(trades[i-1]);
        }

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
        self.fixListItemHeight();

        var pollBalance = function () {
          self.pollAccountBalance(base.asset_id);
          self.pollAccountBalance(quote.asset_id);
        };
        pollBalance();
        setInterval(pollBalance, 7000);

        self.pollCandles(300);
        self.candleInterval = setInterval(function () {
          self.pollCandles(300);
        }, 60000);
        self.handleCandleSwitch();

        self.api.engine.subscribe(base.asset_id + '-' + quote.asset_id, function (msg) {
          self.render(msg);
        });
      }, base.asset_id + '-' + quote.asset_id, offset);
    }, base.asset_id + '-' + quote.asset_id);
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

    $('.trade.form form').submit(function (event) {
      event.preventDefault();
      var form = $(this);
      var data = FormUtils.serialize(this);
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
    $('.charts.container .tabs li').click(function () {
      $('.charts.container .tabs li').removeClass('active');
      $(this).addClass('active');
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
    console.log(msg);
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
        for (var i = 0; i < book.asks.length; i++) {
          self.orderOpenOnPage(book.asks[i]);
        }
        for (var i = 0; i < book.bids.length; i++) {
          self.orderOpenOnPage(book.bids[i]);
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
        self.updateTickerPrice(data.data);
        self.addTradeEntry(data.data);
        self.orderRemoveFromBook(data.data);
        self.orderRemoveFromPage(data.data);
        self.fixListItemHeight();
        break;
    }

    self.renderDepthChart();
  },

  updateTickerPrice: function (o) {
    $('.quote.price').html(parseFloat(o.price));
  },

  addTradeEntry: function (o) {
    const self = this;
    if (self.quote.asset_id === '815b0b1a-2764-3736-8faa-42d694fa620a') {
      o.price = parseFloat(o.price).toFixed(4);
    } else {
      o.price = parseFloat(o.price).toFixed(8);
    }
    o.amount = parseFloat(o.amount).toFixed(4);
    o.sideClass = o.side.toLowerCase();
    o.time = TimeUtils.short(o.created_at);
    $('.history.data').prepend(self.itemTrade(o));
    $('.history.data li:nth-of-type(1n+100)').remove();
  },

  orderOpenOnPage: function (o) {
    const self = this;
    const price = parseFloat(o.price);
    const amount = parseFloat(o.amount);

    o.sideClass = o.side.toLowerCase()
    if (self.quote.asset_id === '815b0b1a-2764-3736-8faa-42d694fa620a') {
      o.price = parseFloat(o.price).toFixed(4);
    } else {
      o.price = parseFloat(o.price).toFixed(8);
    }
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
    const price = parseFloat(o.price);
    const amount = parseFloat(o.amount);

    o.sideClass = o.side.toLowerCase()
    if (self.quote.asset_id === '815b0b1a-2764-3736-8faa-42d694fa620a') {
      o.price = parseFloat(o.price).toFixed(4);
    } else {
      o.price = parseFloat(o.price).toFixed(8);
    }
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
  },

  orderOpenOnBook: function (o) {
    const self = this;
    const price = parseFloat(o.price);
    const amount = parseFloat(o.amount);

    if (o.side === 'ASK') {
      for (var i = 0; i < self.book.asks.length; i++) {
        var bo = self.book.asks[i];
        var bp = parseFloat(bo.price);
        if (bp === price) {
          bo.amount = parseFloat((parseFloat(bo.amount) + amount).toFixed(4));
          return;
        }
        if (bp > price) {
          self.book.asks.splice(i, 0, o);
          return;
        }
      }
      self.book.asks.push(o);
    } else if (o.side === 'BID') {
      for (var i = 0; i < self.book.bids.length; i++) {
        var bo = self.book.bids[i];
        var bp = parseFloat(bo.price);
        if (bp === price) {
          bo.amount = parseFloat((parseFloat(bo.amount) + amount).toFixed(4));
          return;
        }
        if (bp < price) {
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
      if (parseFloat(bo.price) !== price) {
        continue;
      }

      bo.amount = parseFloat((parseFloat(bo.amount) - amount).toFixed(4));
      if (bo.amount === 0) {
        list.splice(i, 1);
      }
      return;
    }
  },

  pollAccountBalance: function (asset) {
    if (this.api.account.token() === '') {
      return;
    }

    const self = this;
    self.api.mixin.asset(function (resp) {
      if (resp.error) {
        return true;
      }

      var data = resp.data;
      $('.balance.' + data.symbol).css({display: 'flex'});
      $('.asset.amount.' + data.symbol).html(data.balance);
    }, asset);
  }
};

export default Market;
