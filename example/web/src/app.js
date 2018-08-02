import 'simple-line-icons/scss/simple-line-icons.scss';
import './layout.scss';
import $ from 'jquery';
import Navigo from 'navigo';
import Locale from './locale';
import API from './api';
import Market from './market';
import Account from './account';

const PartialLoading = require('./loading.html');
const Error404 = require('./404.html');
const router = new Navigo(WEB_ROOT);
const api = new API(router, API_ROOT, ENGINE_ROOT);

window.i18n = new Locale(navigator.language);

router.replace = function(url) {
  this.resolve(url);
  this.pause(true);
  this.navigate(url);
  this.pause(false);
};

router.hooks({
  before: function(done, params) {
    $('body').attr('class', 'loading layout');
    $('#layout-container').html(PartialLoading());
    $('title').html(APP_NAME);
    done(true);
  },
  after: function(params) {
    router.updatePageLinks();
  }
});

router.on({
  '/': function () {
    new Market(router, api).index();
  },
  '/users/new': function () {
    new Account(router, api).signUp();
  },
  '/sessions/new': function () {
    new Account(router, api).signIn();
  },
  '/passwords/new': function () {
    new Account(router, api).resetPassword();
  }
}).notFound(function () {
  $('#layout-container').html(Error404());
  $('body').attr('class', 'error layout');
  router.updatePageLinks();
}).resolve();
