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
}

Market.prototype = {
  index: function (market) {
    if (!market) {
      this.router.replace('/trade/BTC-USDT');
      return;
    }

    var pair = this.api.asset.market(market);
    if (!pair) {
      this.router.replace('/trade/BTC-USDT');
      return;
    }
    const base = pair[0];
    const quote = pair[1];

    const self = this;
    var data = require('./depth.json');
    var bids = data.data.bids;
    var asks = data.data.asks;
    var asksData = [];
    var bidsData = [];
    for (var i = 0; i < bids.length; i++) {
      bidsData.push({
        price: parseFloat(bids[i].price).toFixed(8),
        amount: parseFloat(bids[i].amount).toFixed(4)
      });
    }
    for (var i = asks.length; i > 0; i--) {
      asksData.push({
        price: parseFloat(asks[i-1].price).toFixed(8),
        amount: parseFloat(asks[i-1].amount).toFixed(4)
      });
    }

    self.handlePageScroll(market);

    $('body').attr('class', 'market layout');
    $('#layout-container').html(self.templateIndex({
      logoURL: require('./logo.png'),
      symbolURL: require('./symbol.png')
    })).append(self.templateTrade({
      base: base,
      quote: quote,
      asks: asksData,
      bids: bidsData
    }));

    $('.layout.nav .logo a').click(function() {
      var offset = $('.layout.header').outerHeight() - $('.layout.nav').outerHeight();
      window.scrollTo({ top: offset, behavior: 'smooth' });
    });

    var scroll = $(window).scrollTop();
    var offset = $('.layout.header').outerHeight() + $('.markets.container').outerHeight() - $('.layout.nav').outerHeight() + 1;
    console.log(scroll, offset);
    if (scroll < offset) {
      window.scrollTo({ top: offset, behavior: 'smooth' });
    }

    self.fixListItemHeight();
    self.renderChart(bids, asks);
    self.handleOrderCreate();
    self.handleFormSwitch();
    self.handleBookHistorySwitch();

    self.api.engine.subscribe(base.asset_id + '-' + quote.asset_id, self.render);
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
      data.type = $('.type.tab.active').attr('data-type');
      data.side = $('.side.tab.active').attr('data-side');
      data.trace_id = uuid().toLowerCase();
      data.quote = quote.asset_id;
      data.base = base.asset_id;
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

        console.log(resp);
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
    var total = $('.order.book').height() - $('.order.book .spread').outerHeight() - $('.book.tab').outerHeight();
    var count = parseInt(total / $('.order.book .ask').outerHeight() / 2) * 2;
    var line = (total / count) + 'px';
    $('.order.book .ask').css({'line-height': line, height: line});
    $('.order.book .bid').css({'line-height': line, height: line});
    $('.order.book .header li').css({'line-height': line, height: line});
    $('.order.book .header').css({'top': $('.book.tab').outerHeight()});

    total = $('.trade.history').height() - $('.history.tab').outerHeight();
    count = parseInt(total / $('.trade.history .ask').outerHeight() / 2) * 2;
    line = (total / count) + 'px';
    $('.trade.history .ask').css({'line-height': line, height: line});
    $('.trade.history .bid').css({'line-height': line, height: line});
  },

  renderChart: function (bids, asks) {
    var chart = new Chart();
    chart.renderPrice($('.price.chart')[0]);
    chart.renderDepth($('.depth.chart')[0], bids, asks);
  },

  render: function (msg) {
    console.log(msg);
  }
};

export default Market;
