import $ from 'jquery';

function Chart() {
}

Chart.prototype = {
  prepareCandleData: function (data) {
    var ohlc = [],
      volume = [],
      dataLength = data.length;

    for (var i = 0; i < dataLength; i += 1) {
      ohlc.push([
        data[i][0] * 1000, // the date
        data[i][3], // open
        data[i][2], // high
        data[i][1], // low
        data[i][4] // close
      ]);

      volume.push([
        data[i][0] * 1000, // the date
        data[i][5] // the volume
      ]);
    }

    return [ohlc, volume];
  },

  renderPrice: function (ele, currency, data) {
    var groupingUnits = [
      ['minute', [1, 5, 15, 30]],
      ['hour', [1, 6, 12, 24]]
    ];

    data = this.prepareCandleData(data);
    var ohlc = data[0];
    var volume = data[1];

    var chart = Highcharts.stockChart(ele, {
      chart: {
        zoomType: null,
        panning: false,
        spacing: [0, 0, 0, 0]
      },

      credits: {
        enabled: false
      },

      rangeSelector: {
        enabled: false
      },

      scrollbar: {
        enabled: false
      },

      navigator: {
        enabled: false
      },

      legend: {
        enabled: false
      },

      plotOptions: {
        series: {
          stickyTracking: false,
          showInLegend: false
        }
      },

      yAxis: [{
        labels: {
          align: 'right',
          x: -3
        },
        height: '70%',
        gridLineWidth: 0.5,
        lineWidth: 0
      }, {
        labels: {
          align: 'right',
          x: -3
        },
        top: '71%',
        height: '29%',
        offset: 0,
        gridLineWidth: 0.5,
        lineWidth: 0
      }],

      tooltip: {
        followPointer: true,
        followTouchMove: true,
        split: true
      },

      series: [{
        type: 'column',
        name: 'Volume',
        data: volume,
        yAxis: 1,
        dataGrouping: {
          units: groupingUnits
        },
        color: 'rgba(41,149,242,0.3)'
      }, {
        type: 'candlestick',
        id: 'candle',
        name: currency,
        data: ohlc,
        dataGrouping: {
          units: groupingUnits
        }
      }, {
        type: 'ema',
        linkedTo: 'candle',
        params: {
          period: 12
        },
        color: 'rgba(255,155,100,0.5)',
        lineWidth: 1
      }, {
        type: 'ema',
        linkedTo: 'candle',
        params: {
          period: 26
        },
        color: 'rgba(100,155,255,0.5)',
        lineWidth: 1
      }]
    });

    return chart;
  },

  renderDepth: function (ele, bids, asks) {
    if (bids.length === 0 || asks.length === 0) {
      return undefined;
    }

    var bidsData = [];
    for(var i = 0; i < bids.length; i++) {
      bids[i].volume = parseFloat(bids[i].amount);
      if (i > 0) {
        bids[i].volume = bids[i-1].volume + bids[i].volume;
      }
      bidsData.push({
        x: parseFloat(bids[i].price),
        y: bids[i].volume
      });
    }
    bidsData = bidsData.reverse();

    var asksData = [];
    for(var i = 0; i < asks.length; i++) {
      asks[i].volume = parseFloat(parseFloat(asks[i].amount).toFixed(4));
      if (i > 0) {
        asks[i].volume = parseFloat((asks[i-1].volume + asks[i].volume).toFixed(4));
      }
      asksData.push({
        x: parseFloat(asks[i].price),
        y: asks[i].volume
      });
    }

    var minPrice = bidsData[0].x;
    var maxPrice = asksData[asksData.length-1].x;
    var maxVolume = bidsData[0].y;
    if (asksData[asksData.length-1].y > maxVolume) {
      maxVolume = asksData[asksData.length-1].y;
    }

    var chart = Highcharts.chart(ele, {
      chart: {
        zoomType: null,
        panning: false,
        spacing: [0, 0, 0, 0]
      },

      credits: {
        enabled: false
      },

      rangeSelector: {
        enabled: false
      },

      scrollbar: {
        enabled: false
      },

      navigator: {
        enabled: false
      },

      legend: {
        enabled: false
      },

      title: {
        text: null
      },

      xAxis: {
        gridLineWidth: 0.5,
        min: minPrice,
        max: maxPrice
      },

      yAxis: {
        opposite: true,
        labels: {
          align: 'right',
          x: -3,
          y: -2
        },
        lineWidth: 0,
        resize: {
          enabled: true
        },
        gridLineWidth: 0.5,
        max: maxVolume,
        title: {
          text: '',
        },
      },

      plotOptions: {
        series: {
          stickyTracking: false,
          animation: false,
          marker: {
            enabled: false,
            symbol: 'circle',
            states: {
              hover: {
                enabled: true
              }
            }
          }
        }
      },
      tooltip: {
        followPointer: true,
        followTouchMove: false,
        crosshairs: [true, true],
        formatter: function () {
          return 'Price <b>' + this.x + '</b> <br/>' + this.series.name + '<b>' + this.y + '</b>';
        }
      },
      series: [
        {
          type: 'area',
          name: 'Buy orders',
          data: bidsData,
          color: 'rgba(1,170,120,1.0)',
          fillColor: 'rgba(1,170,120,0.2)'
        },
        {
          type: 'area',
          name: 'Sell orders',
          data: asksData,
          color: 'rgba(255,95,115,1.0)',
          fillColor: 'rgba(255,95,115,0.2)'
        }
      ]
    });

    return chart;
  }
};

export default Chart;
