(() => {
  var __create = Object.create;
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __getProtoOf = Object.getPrototypeOf;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __commonJS = (cb, mod) => function __require() {
    return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
    // If the importer is in node compatibility mode or this is not an ESM
    // file that has been converted to a CommonJS file using a Babel-
    // compatible transform (i.e. "__esModule" has not been set), then set
    // "default" to the CommonJS "module.exports" for node compatibility.
    isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
    mod
  ));

  // node_modules/react/cjs/react.production.min.js
  var require_react_production_min = __commonJS({
    "node_modules/react/cjs/react.production.min.js"(exports) {
      "use strict";
      var l = Symbol.for("react.element");
      var n = Symbol.for("react.portal");
      var p = Symbol.for("react.fragment");
      var q = Symbol.for("react.strict_mode");
      var r2 = Symbol.for("react.profiler");
      var t = Symbol.for("react.provider");
      var u = Symbol.for("react.context");
      var v = Symbol.for("react.forward_ref");
      var w = Symbol.for("react.suspense");
      var x = Symbol.for("react.memo");
      var y = Symbol.for("react.lazy");
      var z = Symbol.iterator;
      function A(a) {
        if (null === a || "object" !== typeof a) return null;
        a = z && a[z] || a["@@iterator"];
        return "function" === typeof a ? a : null;
      }
      var B = { isMounted: function() {
        return false;
      }, enqueueForceUpdate: function() {
      }, enqueueReplaceState: function() {
      }, enqueueSetState: function() {
      } };
      var C = Object.assign;
      var D = {};
      function E(a, b, e) {
        this.props = a;
        this.context = b;
        this.refs = D;
        this.updater = e || B;
      }
      E.prototype.isReactComponent = {};
      E.prototype.setState = function(a, b) {
        if ("object" !== typeof a && "function" !== typeof a && null != a) throw Error("setState(...): takes an object of state variables to update or a function which returns an object of state variables.");
        this.updater.enqueueSetState(this, a, b, "setState");
      };
      E.prototype.forceUpdate = function(a) {
        this.updater.enqueueForceUpdate(this, a, "forceUpdate");
      };
      function F() {
      }
      F.prototype = E.prototype;
      function G(a, b, e) {
        this.props = a;
        this.context = b;
        this.refs = D;
        this.updater = e || B;
      }
      var H = G.prototype = new F();
      H.constructor = G;
      C(H, E.prototype);
      H.isPureReactComponent = true;
      var I = Array.isArray;
      var J = Object.prototype.hasOwnProperty;
      var K = { current: null };
      var L = { key: true, ref: true, __self: true, __source: true };
      function M(a, b, e) {
        var d, c = {}, k = null, h = null;
        if (null != b) for (d in void 0 !== b.ref && (h = b.ref), void 0 !== b.key && (k = "" + b.key), b) J.call(b, d) && !L.hasOwnProperty(d) && (c[d] = b[d]);
        var g = arguments.length - 2;
        if (1 === g) c.children = e;
        else if (1 < g) {
          for (var f = Array(g), m = 0; m < g; m++) f[m] = arguments[m + 2];
          c.children = f;
        }
        if (a && a.defaultProps) for (d in g = a.defaultProps, g) void 0 === c[d] && (c[d] = g[d]);
        return { $$typeof: l, type: a, key: k, ref: h, props: c, _owner: K.current };
      }
      function N(a, b) {
        return { $$typeof: l, type: a.type, key: b, ref: a.ref, props: a.props, _owner: a._owner };
      }
      function O(a) {
        return "object" === typeof a && null !== a && a.$$typeof === l;
      }
      function escape(a) {
        var b = { "=": "=0", ":": "=2" };
        return "$" + a.replace(/[=:]/g, function(a2) {
          return b[a2];
        });
      }
      var P = /\/+/g;
      function Q(a, b) {
        return "object" === typeof a && null !== a && null != a.key ? escape("" + a.key) : b.toString(36);
      }
      function R(a, b, e, d, c) {
        var k = typeof a;
        if ("undefined" === k || "boolean" === k) a = null;
        var h = false;
        if (null === a) h = true;
        else switch (k) {
          case "string":
          case "number":
            h = true;
            break;
          case "object":
            switch (a.$$typeof) {
              case l:
              case n:
                h = true;
            }
        }
        if (h) return h = a, c = c(h), a = "" === d ? "." + Q(h, 0) : d, I(c) ? (e = "", null != a && (e = a.replace(P, "$&/") + "/"), R(c, b, e, "", function(a2) {
          return a2;
        })) : null != c && (O(c) && (c = N(c, e + (!c.key || h && h.key === c.key ? "" : ("" + c.key).replace(P, "$&/") + "/") + a)), b.push(c)), 1;
        h = 0;
        d = "" === d ? "." : d + ":";
        if (I(a)) for (var g = 0; g < a.length; g++) {
          k = a[g];
          var f = d + Q(k, g);
          h += R(k, b, e, f, c);
        }
        else if (f = A(a), "function" === typeof f) for (a = f.call(a), g = 0; !(k = a.next()).done; ) k = k.value, f = d + Q(k, g++), h += R(k, b, e, f, c);
        else if ("object" === k) throw b = String(a), Error("Objects are not valid as a React child (found: " + ("[object Object]" === b ? "object with keys {" + Object.keys(a).join(", ") + "}" : b) + "). If you meant to render a collection of children, use an array instead.");
        return h;
      }
      function S(a, b, e) {
        if (null == a) return a;
        var d = [], c = 0;
        R(a, d, "", "", function(a2) {
          return b.call(e, a2, c++);
        });
        return d;
      }
      function T(a) {
        if (-1 === a._status) {
          var b = a._result;
          b = b();
          b.then(function(b2) {
            if (0 === a._status || -1 === a._status) a._status = 1, a._result = b2;
          }, function(b2) {
            if (0 === a._status || -1 === a._status) a._status = 2, a._result = b2;
          });
          -1 === a._status && (a._status = 0, a._result = b);
        }
        if (1 === a._status) return a._result.default;
        throw a._result;
      }
      var U = { current: null };
      var V = { transition: null };
      var W = { ReactCurrentDispatcher: U, ReactCurrentBatchConfig: V, ReactCurrentOwner: K };
      function X() {
        throw Error("act(...) is not supported in production builds of React.");
      }
      exports.Children = { map: S, forEach: function(a, b, e) {
        S(a, function() {
          b.apply(this, arguments);
        }, e);
      }, count: function(a) {
        var b = 0;
        S(a, function() {
          b++;
        });
        return b;
      }, toArray: function(a) {
        return S(a, function(a2) {
          return a2;
        }) || [];
      }, only: function(a) {
        if (!O(a)) throw Error("React.Children.only expected to receive a single React element child.");
        return a;
      } };
      exports.Component = E;
      exports.Fragment = p;
      exports.Profiler = r2;
      exports.PureComponent = G;
      exports.StrictMode = q;
      exports.Suspense = w;
      exports.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED = W;
      exports.act = X;
      exports.cloneElement = function(a, b, e) {
        if (null === a || void 0 === a) throw Error("React.cloneElement(...): The argument must be a React element, but you passed " + a + ".");
        var d = C({}, a.props), c = a.key, k = a.ref, h = a._owner;
        if (null != b) {
          void 0 !== b.ref && (k = b.ref, h = K.current);
          void 0 !== b.key && (c = "" + b.key);
          if (a.type && a.type.defaultProps) var g = a.type.defaultProps;
          for (f in b) J.call(b, f) && !L.hasOwnProperty(f) && (d[f] = void 0 === b[f] && void 0 !== g ? g[f] : b[f]);
        }
        var f = arguments.length - 2;
        if (1 === f) d.children = e;
        else if (1 < f) {
          g = Array(f);
          for (var m = 0; m < f; m++) g[m] = arguments[m + 2];
          d.children = g;
        }
        return { $$typeof: l, type: a.type, key: c, ref: k, props: d, _owner: h };
      };
      exports.createContext = function(a) {
        a = { $$typeof: u, _currentValue: a, _currentValue2: a, _threadCount: 0, Provider: null, Consumer: null, _defaultValue: null, _globalName: null };
        a.Provider = { $$typeof: t, _context: a };
        return a.Consumer = a;
      };
      exports.createElement = M;
      exports.createFactory = function(a) {
        var b = M.bind(null, a);
        b.type = a;
        return b;
      };
      exports.createRef = function() {
        return { current: null };
      };
      exports.forwardRef = function(a) {
        return { $$typeof: v, render: a };
      };
      exports.isValidElement = O;
      exports.lazy = function(a) {
        return { $$typeof: y, _payload: { _status: -1, _result: a }, _init: T };
      };
      exports.memo = function(a, b) {
        return { $$typeof: x, type: a, compare: void 0 === b ? null : b };
      };
      exports.startTransition = function(a) {
        var b = V.transition;
        V.transition = {};
        try {
          a();
        } finally {
          V.transition = b;
        }
      };
      exports.unstable_act = X;
      exports.useCallback = function(a, b) {
        return U.current.useCallback(a, b);
      };
      exports.useContext = function(a) {
        return U.current.useContext(a);
      };
      exports.useDebugValue = function() {
      };
      exports.useDeferredValue = function(a) {
        return U.current.useDeferredValue(a);
      };
      exports.useEffect = function(a, b) {
        return U.current.useEffect(a, b);
      };
      exports.useId = function() {
        return U.current.useId();
      };
      exports.useImperativeHandle = function(a, b, e) {
        return U.current.useImperativeHandle(a, b, e);
      };
      exports.useInsertionEffect = function(a, b) {
        return U.current.useInsertionEffect(a, b);
      };
      exports.useLayoutEffect = function(a, b) {
        return U.current.useLayoutEffect(a, b);
      };
      exports.useMemo = function(a, b) {
        return U.current.useMemo(a, b);
      };
      exports.useReducer = function(a, b, e) {
        return U.current.useReducer(a, b, e);
      };
      exports.useRef = function(a) {
        return U.current.useRef(a);
      };
      exports.useState = function(a) {
        return U.current.useState(a);
      };
      exports.useSyncExternalStore = function(a, b, e) {
        return U.current.useSyncExternalStore(a, b, e);
      };
      exports.useTransition = function() {
        return U.current.useTransition();
      };
      exports.version = "18.3.1";
    }
  });

  // node_modules/react/index.js
  var require_react = __commonJS({
    "node_modules/react/index.js"(exports, module) {
      "use strict";
      if (true) {
        module.exports = require_react_production_min();
      } else {
        module.exports = null;
      }
    }
  });

  // node_modules/scheduler/cjs/scheduler.production.min.js
  var require_scheduler_production_min = __commonJS({
    "node_modules/scheduler/cjs/scheduler.production.min.js"(exports) {
      "use strict";
      function f(a, b) {
        var c = a.length;
        a.push(b);
        a: for (; 0 < c; ) {
          var d = c - 1 >>> 1, e = a[d];
          if (0 < g(e, b)) a[d] = b, a[c] = e, c = d;
          else break a;
        }
      }
      function h(a) {
        return 0 === a.length ? null : a[0];
      }
      function k(a) {
        if (0 === a.length) return null;
        var b = a[0], c = a.pop();
        if (c !== b) {
          a[0] = c;
          a: for (var d = 0, e = a.length, w = e >>> 1; d < w; ) {
            var m = 2 * (d + 1) - 1, C = a[m], n = m + 1, x = a[n];
            if (0 > g(C, c)) n < e && 0 > g(x, C) ? (a[d] = x, a[n] = c, d = n) : (a[d] = C, a[m] = c, d = m);
            else if (n < e && 0 > g(x, c)) a[d] = x, a[n] = c, d = n;
            else break a;
          }
        }
        return b;
      }
      function g(a, b) {
        var c = a.sortIndex - b.sortIndex;
        return 0 !== c ? c : a.id - b.id;
      }
      if ("object" === typeof performance && "function" === typeof performance.now) {
        l = performance;
        exports.unstable_now = function() {
          return l.now();
        };
      } else {
        p = Date, q = p.now();
        exports.unstable_now = function() {
          return p.now() - q;
        };
      }
      var l;
      var p;
      var q;
      var r2 = [];
      var t = [];
      var u = 1;
      var v = null;
      var y = 3;
      var z = false;
      var A = false;
      var B = false;
      var D = "function" === typeof setTimeout ? setTimeout : null;
      var E = "function" === typeof clearTimeout ? clearTimeout : null;
      var F = "undefined" !== typeof setImmediate ? setImmediate : null;
      "undefined" !== typeof navigator && void 0 !== navigator.scheduling && void 0 !== navigator.scheduling.isInputPending && navigator.scheduling.isInputPending.bind(navigator.scheduling);
      function G(a) {
        for (var b = h(t); null !== b; ) {
          if (null === b.callback) k(t);
          else if (b.startTime <= a) k(t), b.sortIndex = b.expirationTime, f(r2, b);
          else break;
          b = h(t);
        }
      }
      function H(a) {
        B = false;
        G(a);
        if (!A) if (null !== h(r2)) A = true, I(J);
        else {
          var b = h(t);
          null !== b && K(H, b.startTime - a);
        }
      }
      function J(a, b) {
        A = false;
        B && (B = false, E(L), L = -1);
        z = true;
        var c = y;
        try {
          G(b);
          for (v = h(r2); null !== v && (!(v.expirationTime > b) || a && !M()); ) {
            var d = v.callback;
            if ("function" === typeof d) {
              v.callback = null;
              y = v.priorityLevel;
              var e = d(v.expirationTime <= b);
              b = exports.unstable_now();
              "function" === typeof e ? v.callback = e : v === h(r2) && k(r2);
              G(b);
            } else k(r2);
            v = h(r2);
          }
          if (null !== v) var w = true;
          else {
            var m = h(t);
            null !== m && K(H, m.startTime - b);
            w = false;
          }
          return w;
        } finally {
          v = null, y = c, z = false;
        }
      }
      var N = false;
      var O = null;
      var L = -1;
      var P = 5;
      var Q = -1;
      function M() {
        return exports.unstable_now() - Q < P ? false : true;
      }
      function R() {
        if (null !== O) {
          var a = exports.unstable_now();
          Q = a;
          var b = true;
          try {
            b = O(true, a);
          } finally {
            b ? S() : (N = false, O = null);
          }
        } else N = false;
      }
      var S;
      if ("function" === typeof F) S = function() {
        F(R);
      };
      else if ("undefined" !== typeof MessageChannel) {
        T = new MessageChannel(), U = T.port2;
        T.port1.onmessage = R;
        S = function() {
          U.postMessage(null);
        };
      } else S = function() {
        D(R, 0);
      };
      var T;
      var U;
      function I(a) {
        O = a;
        N || (N = true, S());
      }
      function K(a, b) {
        L = D(function() {
          a(exports.unstable_now());
        }, b);
      }
      exports.unstable_IdlePriority = 5;
      exports.unstable_ImmediatePriority = 1;
      exports.unstable_LowPriority = 4;
      exports.unstable_NormalPriority = 3;
      exports.unstable_Profiling = null;
      exports.unstable_UserBlockingPriority = 2;
      exports.unstable_cancelCallback = function(a) {
        a.callback = null;
      };
      exports.unstable_continueExecution = function() {
        A || z || (A = true, I(J));
      };
      exports.unstable_forceFrameRate = function(a) {
        0 > a || 125 < a ? console.error("forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported") : P = 0 < a ? Math.floor(1e3 / a) : 5;
      };
      exports.unstable_getCurrentPriorityLevel = function() {
        return y;
      };
      exports.unstable_getFirstCallbackNode = function() {
        return h(r2);
      };
      exports.unstable_next = function(a) {
        switch (y) {
          case 1:
          case 2:
          case 3:
            var b = 3;
            break;
          default:
            b = y;
        }
        var c = y;
        y = b;
        try {
          return a();
        } finally {
          y = c;
        }
      };
      exports.unstable_pauseExecution = function() {
      };
      exports.unstable_requestPaint = function() {
      };
      exports.unstable_runWithPriority = function(a, b) {
        switch (a) {
          case 1:
          case 2:
          case 3:
          case 4:
          case 5:
            break;
          default:
            a = 3;
        }
        var c = y;
        y = a;
        try {
          return b();
        } finally {
          y = c;
        }
      };
      exports.unstable_scheduleCallback = function(a, b, c) {
        var d = exports.unstable_now();
        "object" === typeof c && null !== c ? (c = c.delay, c = "number" === typeof c && 0 < c ? d + c : d) : c = d;
        switch (a) {
          case 1:
            var e = -1;
            break;
          case 2:
            e = 250;
            break;
          case 5:
            e = 1073741823;
            break;
          case 4:
            e = 1e4;
            break;
          default:
            e = 5e3;
        }
        e = c + e;
        a = { id: u++, callback: b, priorityLevel: a, startTime: c, expirationTime: e, sortIndex: -1 };
        c > d ? (a.sortIndex = c, f(t, a), null === h(r2) && a === h(t) && (B ? (E(L), L = -1) : B = true, K(H, c - d))) : (a.sortIndex = e, f(r2, a), A || z || (A = true, I(J)));
        return a;
      };
      exports.unstable_shouldYield = M;
      exports.unstable_wrapCallback = function(a) {
        var b = y;
        return function() {
          var c = y;
          y = b;
          try {
            return a.apply(this, arguments);
          } finally {
            y = c;
          }
        };
      };
    }
  });

  // node_modules/scheduler/index.js
  var require_scheduler = __commonJS({
    "node_modules/scheduler/index.js"(exports, module) {
      "use strict";
      if (true) {
        module.exports = require_scheduler_production_min();
      } else {
        module.exports = null;
      }
    }
  });

  // node_modules/react-dom/cjs/react-dom.production.min.js
  var require_react_dom_production_min = __commonJS({
    "node_modules/react-dom/cjs/react-dom.production.min.js"(exports) {
      "use strict";
      var aa = require_react();
      var ca = require_scheduler();
      function p(a) {
        for (var b = "https://reactjs.org/docs/error-decoder.html?invariant=" + a, c = 1; c < arguments.length; c++) b += "&args[]=" + encodeURIComponent(arguments[c]);
        return "Minified React error #" + a + "; visit " + b + " for the full message or use the non-minified dev environment for full errors and additional helpful warnings.";
      }
      var da = /* @__PURE__ */ new Set();
      var ea = {};
      function fa(a, b) {
        ha(a, b);
        ha(a + "Capture", b);
      }
      function ha(a, b) {
        ea[a] = b;
        for (a = 0; a < b.length; a++) da.add(b[a]);
      }
      var ia = !("undefined" === typeof window || "undefined" === typeof window.document || "undefined" === typeof window.document.createElement);
      var ja = Object.prototype.hasOwnProperty;
      var ka = /^[:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD][:A-Z_a-z\u00C0-\u00D6\u00D8-\u00F6\u00F8-\u02FF\u0370-\u037D\u037F-\u1FFF\u200C-\u200D\u2070-\u218F\u2C00-\u2FEF\u3001-\uD7FF\uF900-\uFDCF\uFDF0-\uFFFD\-.0-9\u00B7\u0300-\u036F\u203F-\u2040]*$/;
      var la = {};
      var ma = {};
      function oa(a) {
        if (ja.call(ma, a)) return true;
        if (ja.call(la, a)) return false;
        if (ka.test(a)) return ma[a] = true;
        la[a] = true;
        return false;
      }
      function pa(a, b, c, d) {
        if (null !== c && 0 === c.type) return false;
        switch (typeof b) {
          case "function":
          case "symbol":
            return true;
          case "boolean":
            if (d) return false;
            if (null !== c) return !c.acceptsBooleans;
            a = a.toLowerCase().slice(0, 5);
            return "data-" !== a && "aria-" !== a;
          default:
            return false;
        }
      }
      function qa(a, b, c, d) {
        if (null === b || "undefined" === typeof b || pa(a, b, c, d)) return true;
        if (d) return false;
        if (null !== c) switch (c.type) {
          case 3:
            return !b;
          case 4:
            return false === b;
          case 5:
            return isNaN(b);
          case 6:
            return isNaN(b) || 1 > b;
        }
        return false;
      }
      function v(a, b, c, d, e, f, g) {
        this.acceptsBooleans = 2 === b || 3 === b || 4 === b;
        this.attributeName = d;
        this.attributeNamespace = e;
        this.mustUseProperty = c;
        this.propertyName = a;
        this.type = b;
        this.sanitizeURL = f;
        this.removeEmptyString = g;
      }
      var z = {};
      "children dangerouslySetInnerHTML defaultValue defaultChecked innerHTML suppressContentEditableWarning suppressHydrationWarning style".split(" ").forEach(function(a) {
        z[a] = new v(a, 0, false, a, null, false, false);
      });
      [["acceptCharset", "accept-charset"], ["className", "class"], ["htmlFor", "for"], ["httpEquiv", "http-equiv"]].forEach(function(a) {
        var b = a[0];
        z[b] = new v(b, 1, false, a[1], null, false, false);
      });
      ["contentEditable", "draggable", "spellCheck", "value"].forEach(function(a) {
        z[a] = new v(a, 2, false, a.toLowerCase(), null, false, false);
      });
      ["autoReverse", "externalResourcesRequired", "focusable", "preserveAlpha"].forEach(function(a) {
        z[a] = new v(a, 2, false, a, null, false, false);
      });
      "allowFullScreen async autoFocus autoPlay controls default defer disabled disablePictureInPicture disableRemotePlayback formNoValidate hidden loop noModule noValidate open playsInline readOnly required reversed scoped seamless itemScope".split(" ").forEach(function(a) {
        z[a] = new v(a, 3, false, a.toLowerCase(), null, false, false);
      });
      ["checked", "multiple", "muted", "selected"].forEach(function(a) {
        z[a] = new v(a, 3, true, a, null, false, false);
      });
      ["capture", "download"].forEach(function(a) {
        z[a] = new v(a, 4, false, a, null, false, false);
      });
      ["cols", "rows", "size", "span"].forEach(function(a) {
        z[a] = new v(a, 6, false, a, null, false, false);
      });
      ["rowSpan", "start"].forEach(function(a) {
        z[a] = new v(a, 5, false, a.toLowerCase(), null, false, false);
      });
      var ra = /[\-:]([a-z])/g;
      function sa(a) {
        return a[1].toUpperCase();
      }
      "accent-height alignment-baseline arabic-form baseline-shift cap-height clip-path clip-rule color-interpolation color-interpolation-filters color-profile color-rendering dominant-baseline enable-background fill-opacity fill-rule flood-color flood-opacity font-family font-size font-size-adjust font-stretch font-style font-variant font-weight glyph-name glyph-orientation-horizontal glyph-orientation-vertical horiz-adv-x horiz-origin-x image-rendering letter-spacing lighting-color marker-end marker-mid marker-start overline-position overline-thickness paint-order panose-1 pointer-events rendering-intent shape-rendering stop-color stop-opacity strikethrough-position strikethrough-thickness stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-miterlimit stroke-opacity stroke-width text-anchor text-decoration text-rendering underline-position underline-thickness unicode-bidi unicode-range units-per-em v-alphabetic v-hanging v-ideographic v-mathematical vector-effect vert-adv-y vert-origin-x vert-origin-y word-spacing writing-mode xmlns:xlink x-height".split(" ").forEach(function(a) {
        var b = a.replace(
          ra,
          sa
        );
        z[b] = new v(b, 1, false, a, null, false, false);
      });
      "xlink:actuate xlink:arcrole xlink:role xlink:show xlink:title xlink:type".split(" ").forEach(function(a) {
        var b = a.replace(ra, sa);
        z[b] = new v(b, 1, false, a, "http://www.w3.org/1999/xlink", false, false);
      });
      ["xml:base", "xml:lang", "xml:space"].forEach(function(a) {
        var b = a.replace(ra, sa);
        z[b] = new v(b, 1, false, a, "http://www.w3.org/XML/1998/namespace", false, false);
      });
      ["tabIndex", "crossOrigin"].forEach(function(a) {
        z[a] = new v(a, 1, false, a.toLowerCase(), null, false, false);
      });
      z.xlinkHref = new v("xlinkHref", 1, false, "xlink:href", "http://www.w3.org/1999/xlink", true, false);
      ["src", "href", "action", "formAction"].forEach(function(a) {
        z[a] = new v(a, 1, false, a.toLowerCase(), null, true, true);
      });
      function ta(a, b, c, d) {
        var e = z.hasOwnProperty(b) ? z[b] : null;
        if (null !== e ? 0 !== e.type : d || !(2 < b.length) || "o" !== b[0] && "O" !== b[0] || "n" !== b[1] && "N" !== b[1]) qa(b, c, e, d) && (c = null), d || null === e ? oa(b) && (null === c ? a.removeAttribute(b) : a.setAttribute(b, "" + c)) : e.mustUseProperty ? a[e.propertyName] = null === c ? 3 === e.type ? false : "" : c : (b = e.attributeName, d = e.attributeNamespace, null === c ? a.removeAttribute(b) : (e = e.type, c = 3 === e || 4 === e && true === c ? "" : "" + c, d ? a.setAttributeNS(d, b, c) : a.setAttribute(b, c)));
      }
      var ua = aa.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED;
      var va = Symbol.for("react.element");
      var wa = Symbol.for("react.portal");
      var ya = Symbol.for("react.fragment");
      var za = Symbol.for("react.strict_mode");
      var Aa = Symbol.for("react.profiler");
      var Ba = Symbol.for("react.provider");
      var Ca = Symbol.for("react.context");
      var Da = Symbol.for("react.forward_ref");
      var Ea = Symbol.for("react.suspense");
      var Fa = Symbol.for("react.suspense_list");
      var Ga = Symbol.for("react.memo");
      var Ha = Symbol.for("react.lazy");
      Symbol.for("react.scope");
      Symbol.for("react.debug_trace_mode");
      var Ia = Symbol.for("react.offscreen");
      Symbol.for("react.legacy_hidden");
      Symbol.for("react.cache");
      Symbol.for("react.tracing_marker");
      var Ja = Symbol.iterator;
      function Ka(a) {
        if (null === a || "object" !== typeof a) return null;
        a = Ja && a[Ja] || a["@@iterator"];
        return "function" === typeof a ? a : null;
      }
      var A = Object.assign;
      var La;
      function Ma(a) {
        if (void 0 === La) try {
          throw Error();
        } catch (c) {
          var b = c.stack.trim().match(/\n( *(at )?)/);
          La = b && b[1] || "";
        }
        return "\n" + La + a;
      }
      var Na = false;
      function Oa(a, b) {
        if (!a || Na) return "";
        Na = true;
        var c = Error.prepareStackTrace;
        Error.prepareStackTrace = void 0;
        try {
          if (b) if (b = function() {
            throw Error();
          }, Object.defineProperty(b.prototype, "props", { set: function() {
            throw Error();
          } }), "object" === typeof Reflect && Reflect.construct) {
            try {
              Reflect.construct(b, []);
            } catch (l) {
              var d = l;
            }
            Reflect.construct(a, [], b);
          } else {
            try {
              b.call();
            } catch (l) {
              d = l;
            }
            a.call(b.prototype);
          }
          else {
            try {
              throw Error();
            } catch (l) {
              d = l;
            }
            a();
          }
        } catch (l) {
          if (l && d && "string" === typeof l.stack) {
            for (var e = l.stack.split("\n"), f = d.stack.split("\n"), g = e.length - 1, h = f.length - 1; 1 <= g && 0 <= h && e[g] !== f[h]; ) h--;
            for (; 1 <= g && 0 <= h; g--, h--) if (e[g] !== f[h]) {
              if (1 !== g || 1 !== h) {
                do
                  if (g--, h--, 0 > h || e[g] !== f[h]) {
                    var k = "\n" + e[g].replace(" at new ", " at ");
                    a.displayName && k.includes("<anonymous>") && (k = k.replace("<anonymous>", a.displayName));
                    return k;
                  }
                while (1 <= g && 0 <= h);
              }
              break;
            }
          }
        } finally {
          Na = false, Error.prepareStackTrace = c;
        }
        return (a = a ? a.displayName || a.name : "") ? Ma(a) : "";
      }
      function Pa(a) {
        switch (a.tag) {
          case 5:
            return Ma(a.type);
          case 16:
            return Ma("Lazy");
          case 13:
            return Ma("Suspense");
          case 19:
            return Ma("SuspenseList");
          case 0:
          case 2:
          case 15:
            return a = Oa(a.type, false), a;
          case 11:
            return a = Oa(a.type.render, false), a;
          case 1:
            return a = Oa(a.type, true), a;
          default:
            return "";
        }
      }
      function Qa(a) {
        if (null == a) return null;
        if ("function" === typeof a) return a.displayName || a.name || null;
        if ("string" === typeof a) return a;
        switch (a) {
          case ya:
            return "Fragment";
          case wa:
            return "Portal";
          case Aa:
            return "Profiler";
          case za:
            return "StrictMode";
          case Ea:
            return "Suspense";
          case Fa:
            return "SuspenseList";
        }
        if ("object" === typeof a) switch (a.$$typeof) {
          case Ca:
            return (a.displayName || "Context") + ".Consumer";
          case Ba:
            return (a._context.displayName || "Context") + ".Provider";
          case Da:
            var b = a.render;
            a = a.displayName;
            a || (a = b.displayName || b.name || "", a = "" !== a ? "ForwardRef(" + a + ")" : "ForwardRef");
            return a;
          case Ga:
            return b = a.displayName || null, null !== b ? b : Qa(a.type) || "Memo";
          case Ha:
            b = a._payload;
            a = a._init;
            try {
              return Qa(a(b));
            } catch (c) {
            }
        }
        return null;
      }
      function Ra(a) {
        var b = a.type;
        switch (a.tag) {
          case 24:
            return "Cache";
          case 9:
            return (b.displayName || "Context") + ".Consumer";
          case 10:
            return (b._context.displayName || "Context") + ".Provider";
          case 18:
            return "DehydratedFragment";
          case 11:
            return a = b.render, a = a.displayName || a.name || "", b.displayName || ("" !== a ? "ForwardRef(" + a + ")" : "ForwardRef");
          case 7:
            return "Fragment";
          case 5:
            return b;
          case 4:
            return "Portal";
          case 3:
            return "Root";
          case 6:
            return "Text";
          case 16:
            return Qa(b);
          case 8:
            return b === za ? "StrictMode" : "Mode";
          case 22:
            return "Offscreen";
          case 12:
            return "Profiler";
          case 21:
            return "Scope";
          case 13:
            return "Suspense";
          case 19:
            return "SuspenseList";
          case 25:
            return "TracingMarker";
          case 1:
          case 0:
          case 17:
          case 2:
          case 14:
          case 15:
            if ("function" === typeof b) return b.displayName || b.name || null;
            if ("string" === typeof b) return b;
        }
        return null;
      }
      function Sa(a) {
        switch (typeof a) {
          case "boolean":
          case "number":
          case "string":
          case "undefined":
            return a;
          case "object":
            return a;
          default:
            return "";
        }
      }
      function Ta(a) {
        var b = a.type;
        return (a = a.nodeName) && "input" === a.toLowerCase() && ("checkbox" === b || "radio" === b);
      }
      function Ua(a) {
        var b = Ta(a) ? "checked" : "value", c = Object.getOwnPropertyDescriptor(a.constructor.prototype, b), d = "" + a[b];
        if (!a.hasOwnProperty(b) && "undefined" !== typeof c && "function" === typeof c.get && "function" === typeof c.set) {
          var e = c.get, f = c.set;
          Object.defineProperty(a, b, { configurable: true, get: function() {
            return e.call(this);
          }, set: function(a2) {
            d = "" + a2;
            f.call(this, a2);
          } });
          Object.defineProperty(a, b, { enumerable: c.enumerable });
          return { getValue: function() {
            return d;
          }, setValue: function(a2) {
            d = "" + a2;
          }, stopTracking: function() {
            a._valueTracker = null;
            delete a[b];
          } };
        }
      }
      function Va(a) {
        a._valueTracker || (a._valueTracker = Ua(a));
      }
      function Wa(a) {
        if (!a) return false;
        var b = a._valueTracker;
        if (!b) return true;
        var c = b.getValue();
        var d = "";
        a && (d = Ta(a) ? a.checked ? "true" : "false" : a.value);
        a = d;
        return a !== c ? (b.setValue(a), true) : false;
      }
      function Xa(a) {
        a = a || ("undefined" !== typeof document ? document : void 0);
        if ("undefined" === typeof a) return null;
        try {
          return a.activeElement || a.body;
        } catch (b) {
          return a.body;
        }
      }
      function Ya(a, b) {
        var c = b.checked;
        return A({}, b, { defaultChecked: void 0, defaultValue: void 0, value: void 0, checked: null != c ? c : a._wrapperState.initialChecked });
      }
      function Za(a, b) {
        var c = null == b.defaultValue ? "" : b.defaultValue, d = null != b.checked ? b.checked : b.defaultChecked;
        c = Sa(null != b.value ? b.value : c);
        a._wrapperState = { initialChecked: d, initialValue: c, controlled: "checkbox" === b.type || "radio" === b.type ? null != b.checked : null != b.value };
      }
      function ab(a, b) {
        b = b.checked;
        null != b && ta(a, "checked", b, false);
      }
      function bb(a, b) {
        ab(a, b);
        var c = Sa(b.value), d = b.type;
        if (null != c) if ("number" === d) {
          if (0 === c && "" === a.value || a.value != c) a.value = "" + c;
        } else a.value !== "" + c && (a.value = "" + c);
        else if ("submit" === d || "reset" === d) {
          a.removeAttribute("value");
          return;
        }
        b.hasOwnProperty("value") ? cb(a, b.type, c) : b.hasOwnProperty("defaultValue") && cb(a, b.type, Sa(b.defaultValue));
        null == b.checked && null != b.defaultChecked && (a.defaultChecked = !!b.defaultChecked);
      }
      function db(a, b, c) {
        if (b.hasOwnProperty("value") || b.hasOwnProperty("defaultValue")) {
          var d = b.type;
          if (!("submit" !== d && "reset" !== d || void 0 !== b.value && null !== b.value)) return;
          b = "" + a._wrapperState.initialValue;
          c || b === a.value || (a.value = b);
          a.defaultValue = b;
        }
        c = a.name;
        "" !== c && (a.name = "");
        a.defaultChecked = !!a._wrapperState.initialChecked;
        "" !== c && (a.name = c);
      }
      function cb(a, b, c) {
        if ("number" !== b || Xa(a.ownerDocument) !== a) null == c ? a.defaultValue = "" + a._wrapperState.initialValue : a.defaultValue !== "" + c && (a.defaultValue = "" + c);
      }
      var eb = Array.isArray;
      function fb(a, b, c, d) {
        a = a.options;
        if (b) {
          b = {};
          for (var e = 0; e < c.length; e++) b["$" + c[e]] = true;
          for (c = 0; c < a.length; c++) e = b.hasOwnProperty("$" + a[c].value), a[c].selected !== e && (a[c].selected = e), e && d && (a[c].defaultSelected = true);
        } else {
          c = "" + Sa(c);
          b = null;
          for (e = 0; e < a.length; e++) {
            if (a[e].value === c) {
              a[e].selected = true;
              d && (a[e].defaultSelected = true);
              return;
            }
            null !== b || a[e].disabled || (b = a[e]);
          }
          null !== b && (b.selected = true);
        }
      }
      function gb(a, b) {
        if (null != b.dangerouslySetInnerHTML) throw Error(p(91));
        return A({}, b, { value: void 0, defaultValue: void 0, children: "" + a._wrapperState.initialValue });
      }
      function hb(a, b) {
        var c = b.value;
        if (null == c) {
          c = b.children;
          b = b.defaultValue;
          if (null != c) {
            if (null != b) throw Error(p(92));
            if (eb(c)) {
              if (1 < c.length) throw Error(p(93));
              c = c[0];
            }
            b = c;
          }
          null == b && (b = "");
          c = b;
        }
        a._wrapperState = { initialValue: Sa(c) };
      }
      function ib(a, b) {
        var c = Sa(b.value), d = Sa(b.defaultValue);
        null != c && (c = "" + c, c !== a.value && (a.value = c), null == b.defaultValue && a.defaultValue !== c && (a.defaultValue = c));
        null != d && (a.defaultValue = "" + d);
      }
      function jb(a) {
        var b = a.textContent;
        b === a._wrapperState.initialValue && "" !== b && null !== b && (a.value = b);
      }
      function kb(a) {
        switch (a) {
          case "svg":
            return "http://www.w3.org/2000/svg";
          case "math":
            return "http://www.w3.org/1998/Math/MathML";
          default:
            return "http://www.w3.org/1999/xhtml";
        }
      }
      function lb(a, b) {
        return null == a || "http://www.w3.org/1999/xhtml" === a ? kb(b) : "http://www.w3.org/2000/svg" === a && "foreignObject" === b ? "http://www.w3.org/1999/xhtml" : a;
      }
      var mb;
      var nb = (function(a) {
        return "undefined" !== typeof MSApp && MSApp.execUnsafeLocalFunction ? function(b, c, d, e) {
          MSApp.execUnsafeLocalFunction(function() {
            return a(b, c, d, e);
          });
        } : a;
      })(function(a, b) {
        if ("http://www.w3.org/2000/svg" !== a.namespaceURI || "innerHTML" in a) a.innerHTML = b;
        else {
          mb = mb || document.createElement("div");
          mb.innerHTML = "<svg>" + b.valueOf().toString() + "</svg>";
          for (b = mb.firstChild; a.firstChild; ) a.removeChild(a.firstChild);
          for (; b.firstChild; ) a.appendChild(b.firstChild);
        }
      });
      function ob(a, b) {
        if (b) {
          var c = a.firstChild;
          if (c && c === a.lastChild && 3 === c.nodeType) {
            c.nodeValue = b;
            return;
          }
        }
        a.textContent = b;
      }
      var pb = {
        animationIterationCount: true,
        aspectRatio: true,
        borderImageOutset: true,
        borderImageSlice: true,
        borderImageWidth: true,
        boxFlex: true,
        boxFlexGroup: true,
        boxOrdinalGroup: true,
        columnCount: true,
        columns: true,
        flex: true,
        flexGrow: true,
        flexPositive: true,
        flexShrink: true,
        flexNegative: true,
        flexOrder: true,
        gridArea: true,
        gridRow: true,
        gridRowEnd: true,
        gridRowSpan: true,
        gridRowStart: true,
        gridColumn: true,
        gridColumnEnd: true,
        gridColumnSpan: true,
        gridColumnStart: true,
        fontWeight: true,
        lineClamp: true,
        lineHeight: true,
        opacity: true,
        order: true,
        orphans: true,
        tabSize: true,
        widows: true,
        zIndex: true,
        zoom: true,
        fillOpacity: true,
        floodOpacity: true,
        stopOpacity: true,
        strokeDasharray: true,
        strokeDashoffset: true,
        strokeMiterlimit: true,
        strokeOpacity: true,
        strokeWidth: true
      };
      var qb = ["Webkit", "ms", "Moz", "O"];
      Object.keys(pb).forEach(function(a) {
        qb.forEach(function(b) {
          b = b + a.charAt(0).toUpperCase() + a.substring(1);
          pb[b] = pb[a];
        });
      });
      function rb(a, b, c) {
        return null == b || "boolean" === typeof b || "" === b ? "" : c || "number" !== typeof b || 0 === b || pb.hasOwnProperty(a) && pb[a] ? ("" + b).trim() : b + "px";
      }
      function sb(a, b) {
        a = a.style;
        for (var c in b) if (b.hasOwnProperty(c)) {
          var d = 0 === c.indexOf("--"), e = rb(c, b[c], d);
          "float" === c && (c = "cssFloat");
          d ? a.setProperty(c, e) : a[c] = e;
        }
      }
      var tb = A({ menuitem: true }, { area: true, base: true, br: true, col: true, embed: true, hr: true, img: true, input: true, keygen: true, link: true, meta: true, param: true, source: true, track: true, wbr: true });
      function ub(a, b) {
        if (b) {
          if (tb[a] && (null != b.children || null != b.dangerouslySetInnerHTML)) throw Error(p(137, a));
          if (null != b.dangerouslySetInnerHTML) {
            if (null != b.children) throw Error(p(60));
            if ("object" !== typeof b.dangerouslySetInnerHTML || !("__html" in b.dangerouslySetInnerHTML)) throw Error(p(61));
          }
          if (null != b.style && "object" !== typeof b.style) throw Error(p(62));
        }
      }
      function vb(a, b) {
        if (-1 === a.indexOf("-")) return "string" === typeof b.is;
        switch (a) {
          case "annotation-xml":
          case "color-profile":
          case "font-face":
          case "font-face-src":
          case "font-face-uri":
          case "font-face-format":
          case "font-face-name":
          case "missing-glyph":
            return false;
          default:
            return true;
        }
      }
      var wb = null;
      function xb(a) {
        a = a.target || a.srcElement || window;
        a.correspondingUseElement && (a = a.correspondingUseElement);
        return 3 === a.nodeType ? a.parentNode : a;
      }
      var yb = null;
      var zb = null;
      var Ab = null;
      function Bb(a) {
        if (a = Cb(a)) {
          if ("function" !== typeof yb) throw Error(p(280));
          var b = a.stateNode;
          b && (b = Db(b), yb(a.stateNode, a.type, b));
        }
      }
      function Eb(a) {
        zb ? Ab ? Ab.push(a) : Ab = [a] : zb = a;
      }
      function Fb() {
        if (zb) {
          var a = zb, b = Ab;
          Ab = zb = null;
          Bb(a);
          if (b) for (a = 0; a < b.length; a++) Bb(b[a]);
        }
      }
      function Gb(a, b) {
        return a(b);
      }
      function Hb() {
      }
      var Ib = false;
      function Jb(a, b, c) {
        if (Ib) return a(b, c);
        Ib = true;
        try {
          return Gb(a, b, c);
        } finally {
          if (Ib = false, null !== zb || null !== Ab) Hb(), Fb();
        }
      }
      function Kb(a, b) {
        var c = a.stateNode;
        if (null === c) return null;
        var d = Db(c);
        if (null === d) return null;
        c = d[b];
        a: switch (b) {
          case "onClick":
          case "onClickCapture":
          case "onDoubleClick":
          case "onDoubleClickCapture":
          case "onMouseDown":
          case "onMouseDownCapture":
          case "onMouseMove":
          case "onMouseMoveCapture":
          case "onMouseUp":
          case "onMouseUpCapture":
          case "onMouseEnter":
            (d = !d.disabled) || (a = a.type, d = !("button" === a || "input" === a || "select" === a || "textarea" === a));
            a = !d;
            break a;
          default:
            a = false;
        }
        if (a) return null;
        if (c && "function" !== typeof c) throw Error(p(231, b, typeof c));
        return c;
      }
      var Lb = false;
      if (ia) try {
        Mb = {};
        Object.defineProperty(Mb, "passive", { get: function() {
          Lb = true;
        } });
        window.addEventListener("test", Mb, Mb);
        window.removeEventListener("test", Mb, Mb);
      } catch (a) {
        Lb = false;
      }
      var Mb;
      function Nb(a, b, c, d, e, f, g, h, k) {
        var l = Array.prototype.slice.call(arguments, 3);
        try {
          b.apply(c, l);
        } catch (m) {
          this.onError(m);
        }
      }
      var Ob = false;
      var Pb = null;
      var Qb = false;
      var Rb = null;
      var Sb = { onError: function(a) {
        Ob = true;
        Pb = a;
      } };
      function Tb(a, b, c, d, e, f, g, h, k) {
        Ob = false;
        Pb = null;
        Nb.apply(Sb, arguments);
      }
      function Ub(a, b, c, d, e, f, g, h, k) {
        Tb.apply(this, arguments);
        if (Ob) {
          if (Ob) {
            var l = Pb;
            Ob = false;
            Pb = null;
          } else throw Error(p(198));
          Qb || (Qb = true, Rb = l);
        }
      }
      function Vb(a) {
        var b = a, c = a;
        if (a.alternate) for (; b.return; ) b = b.return;
        else {
          a = b;
          do
            b = a, 0 !== (b.flags & 4098) && (c = b.return), a = b.return;
          while (a);
        }
        return 3 === b.tag ? c : null;
      }
      function Wb(a) {
        if (13 === a.tag) {
          var b = a.memoizedState;
          null === b && (a = a.alternate, null !== a && (b = a.memoizedState));
          if (null !== b) return b.dehydrated;
        }
        return null;
      }
      function Xb(a) {
        if (Vb(a) !== a) throw Error(p(188));
      }
      function Yb(a) {
        var b = a.alternate;
        if (!b) {
          b = Vb(a);
          if (null === b) throw Error(p(188));
          return b !== a ? null : a;
        }
        for (var c = a, d = b; ; ) {
          var e = c.return;
          if (null === e) break;
          var f = e.alternate;
          if (null === f) {
            d = e.return;
            if (null !== d) {
              c = d;
              continue;
            }
            break;
          }
          if (e.child === f.child) {
            for (f = e.child; f; ) {
              if (f === c) return Xb(e), a;
              if (f === d) return Xb(e), b;
              f = f.sibling;
            }
            throw Error(p(188));
          }
          if (c.return !== d.return) c = e, d = f;
          else {
            for (var g = false, h = e.child; h; ) {
              if (h === c) {
                g = true;
                c = e;
                d = f;
                break;
              }
              if (h === d) {
                g = true;
                d = e;
                c = f;
                break;
              }
              h = h.sibling;
            }
            if (!g) {
              for (h = f.child; h; ) {
                if (h === c) {
                  g = true;
                  c = f;
                  d = e;
                  break;
                }
                if (h === d) {
                  g = true;
                  d = f;
                  c = e;
                  break;
                }
                h = h.sibling;
              }
              if (!g) throw Error(p(189));
            }
          }
          if (c.alternate !== d) throw Error(p(190));
        }
        if (3 !== c.tag) throw Error(p(188));
        return c.stateNode.current === c ? a : b;
      }
      function Zb(a) {
        a = Yb(a);
        return null !== a ? $b(a) : null;
      }
      function $b(a) {
        if (5 === a.tag || 6 === a.tag) return a;
        for (a = a.child; null !== a; ) {
          var b = $b(a);
          if (null !== b) return b;
          a = a.sibling;
        }
        return null;
      }
      var ac = ca.unstable_scheduleCallback;
      var bc = ca.unstable_cancelCallback;
      var cc = ca.unstable_shouldYield;
      var dc = ca.unstable_requestPaint;
      var B = ca.unstable_now;
      var ec = ca.unstable_getCurrentPriorityLevel;
      var fc = ca.unstable_ImmediatePriority;
      var gc = ca.unstable_UserBlockingPriority;
      var hc = ca.unstable_NormalPriority;
      var ic = ca.unstable_LowPriority;
      var jc = ca.unstable_IdlePriority;
      var kc = null;
      var lc = null;
      function mc(a) {
        if (lc && "function" === typeof lc.onCommitFiberRoot) try {
          lc.onCommitFiberRoot(kc, a, void 0, 128 === (a.current.flags & 128));
        } catch (b) {
        }
      }
      var oc = Math.clz32 ? Math.clz32 : nc;
      var pc = Math.log;
      var qc = Math.LN2;
      function nc(a) {
        a >>>= 0;
        return 0 === a ? 32 : 31 - (pc(a) / qc | 0) | 0;
      }
      var rc = 64;
      var sc = 4194304;
      function tc(a) {
        switch (a & -a) {
          case 1:
            return 1;
          case 2:
            return 2;
          case 4:
            return 4;
          case 8:
            return 8;
          case 16:
            return 16;
          case 32:
            return 32;
          case 64:
          case 128:
          case 256:
          case 512:
          case 1024:
          case 2048:
          case 4096:
          case 8192:
          case 16384:
          case 32768:
          case 65536:
          case 131072:
          case 262144:
          case 524288:
          case 1048576:
          case 2097152:
            return a & 4194240;
          case 4194304:
          case 8388608:
          case 16777216:
          case 33554432:
          case 67108864:
            return a & 130023424;
          case 134217728:
            return 134217728;
          case 268435456:
            return 268435456;
          case 536870912:
            return 536870912;
          case 1073741824:
            return 1073741824;
          default:
            return a;
        }
      }
      function uc(a, b) {
        var c = a.pendingLanes;
        if (0 === c) return 0;
        var d = 0, e = a.suspendedLanes, f = a.pingedLanes, g = c & 268435455;
        if (0 !== g) {
          var h = g & ~e;
          0 !== h ? d = tc(h) : (f &= g, 0 !== f && (d = tc(f)));
        } else g = c & ~e, 0 !== g ? d = tc(g) : 0 !== f && (d = tc(f));
        if (0 === d) return 0;
        if (0 !== b && b !== d && 0 === (b & e) && (e = d & -d, f = b & -b, e >= f || 16 === e && 0 !== (f & 4194240))) return b;
        0 !== (d & 4) && (d |= c & 16);
        b = a.entangledLanes;
        if (0 !== b) for (a = a.entanglements, b &= d; 0 < b; ) c = 31 - oc(b), e = 1 << c, d |= a[c], b &= ~e;
        return d;
      }
      function vc(a, b) {
        switch (a) {
          case 1:
          case 2:
          case 4:
            return b + 250;
          case 8:
          case 16:
          case 32:
          case 64:
          case 128:
          case 256:
          case 512:
          case 1024:
          case 2048:
          case 4096:
          case 8192:
          case 16384:
          case 32768:
          case 65536:
          case 131072:
          case 262144:
          case 524288:
          case 1048576:
          case 2097152:
            return b + 5e3;
          case 4194304:
          case 8388608:
          case 16777216:
          case 33554432:
          case 67108864:
            return -1;
          case 134217728:
          case 268435456:
          case 536870912:
          case 1073741824:
            return -1;
          default:
            return -1;
        }
      }
      function wc(a, b) {
        for (var c = a.suspendedLanes, d = a.pingedLanes, e = a.expirationTimes, f = a.pendingLanes; 0 < f; ) {
          var g = 31 - oc(f), h = 1 << g, k = e[g];
          if (-1 === k) {
            if (0 === (h & c) || 0 !== (h & d)) e[g] = vc(h, b);
          } else k <= b && (a.expiredLanes |= h);
          f &= ~h;
        }
      }
      function xc(a) {
        a = a.pendingLanes & -1073741825;
        return 0 !== a ? a : a & 1073741824 ? 1073741824 : 0;
      }
      function yc() {
        var a = rc;
        rc <<= 1;
        0 === (rc & 4194240) && (rc = 64);
        return a;
      }
      function zc(a) {
        for (var b = [], c = 0; 31 > c; c++) b.push(a);
        return b;
      }
      function Ac(a, b, c) {
        a.pendingLanes |= b;
        536870912 !== b && (a.suspendedLanes = 0, a.pingedLanes = 0);
        a = a.eventTimes;
        b = 31 - oc(b);
        a[b] = c;
      }
      function Bc(a, b) {
        var c = a.pendingLanes & ~b;
        a.pendingLanes = b;
        a.suspendedLanes = 0;
        a.pingedLanes = 0;
        a.expiredLanes &= b;
        a.mutableReadLanes &= b;
        a.entangledLanes &= b;
        b = a.entanglements;
        var d = a.eventTimes;
        for (a = a.expirationTimes; 0 < c; ) {
          var e = 31 - oc(c), f = 1 << e;
          b[e] = 0;
          d[e] = -1;
          a[e] = -1;
          c &= ~f;
        }
      }
      function Cc(a, b) {
        var c = a.entangledLanes |= b;
        for (a = a.entanglements; c; ) {
          var d = 31 - oc(c), e = 1 << d;
          e & b | a[d] & b && (a[d] |= b);
          c &= ~e;
        }
      }
      var C = 0;
      function Dc(a) {
        a &= -a;
        return 1 < a ? 4 < a ? 0 !== (a & 268435455) ? 16 : 536870912 : 4 : 1;
      }
      var Ec;
      var Fc;
      var Gc;
      var Hc;
      var Ic;
      var Jc = false;
      var Kc = [];
      var Lc = null;
      var Mc = null;
      var Nc = null;
      var Oc = /* @__PURE__ */ new Map();
      var Pc = /* @__PURE__ */ new Map();
      var Qc = [];
      var Rc = "mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset submit".split(" ");
      function Sc(a, b) {
        switch (a) {
          case "focusin":
          case "focusout":
            Lc = null;
            break;
          case "dragenter":
          case "dragleave":
            Mc = null;
            break;
          case "mouseover":
          case "mouseout":
            Nc = null;
            break;
          case "pointerover":
          case "pointerout":
            Oc.delete(b.pointerId);
            break;
          case "gotpointercapture":
          case "lostpointercapture":
            Pc.delete(b.pointerId);
        }
      }
      function Tc(a, b, c, d, e, f) {
        if (null === a || a.nativeEvent !== f) return a = { blockedOn: b, domEventName: c, eventSystemFlags: d, nativeEvent: f, targetContainers: [e] }, null !== b && (b = Cb(b), null !== b && Fc(b)), a;
        a.eventSystemFlags |= d;
        b = a.targetContainers;
        null !== e && -1 === b.indexOf(e) && b.push(e);
        return a;
      }
      function Uc(a, b, c, d, e) {
        switch (b) {
          case "focusin":
            return Lc = Tc(Lc, a, b, c, d, e), true;
          case "dragenter":
            return Mc = Tc(Mc, a, b, c, d, e), true;
          case "mouseover":
            return Nc = Tc(Nc, a, b, c, d, e), true;
          case "pointerover":
            var f = e.pointerId;
            Oc.set(f, Tc(Oc.get(f) || null, a, b, c, d, e));
            return true;
          case "gotpointercapture":
            return f = e.pointerId, Pc.set(f, Tc(Pc.get(f) || null, a, b, c, d, e)), true;
        }
        return false;
      }
      function Vc(a) {
        var b = Wc(a.target);
        if (null !== b) {
          var c = Vb(b);
          if (null !== c) {
            if (b = c.tag, 13 === b) {
              if (b = Wb(c), null !== b) {
                a.blockedOn = b;
                Ic(a.priority, function() {
                  Gc(c);
                });
                return;
              }
            } else if (3 === b && c.stateNode.current.memoizedState.isDehydrated) {
              a.blockedOn = 3 === c.tag ? c.stateNode.containerInfo : null;
              return;
            }
          }
        }
        a.blockedOn = null;
      }
      function Xc(a) {
        if (null !== a.blockedOn) return false;
        for (var b = a.targetContainers; 0 < b.length; ) {
          var c = Yc(a.domEventName, a.eventSystemFlags, b[0], a.nativeEvent);
          if (null === c) {
            c = a.nativeEvent;
            var d = new c.constructor(c.type, c);
            wb = d;
            c.target.dispatchEvent(d);
            wb = null;
          } else return b = Cb(c), null !== b && Fc(b), a.blockedOn = c, false;
          b.shift();
        }
        return true;
      }
      function Zc(a, b, c) {
        Xc(a) && c.delete(b);
      }
      function $c() {
        Jc = false;
        null !== Lc && Xc(Lc) && (Lc = null);
        null !== Mc && Xc(Mc) && (Mc = null);
        null !== Nc && Xc(Nc) && (Nc = null);
        Oc.forEach(Zc);
        Pc.forEach(Zc);
      }
      function ad(a, b) {
        a.blockedOn === b && (a.blockedOn = null, Jc || (Jc = true, ca.unstable_scheduleCallback(ca.unstable_NormalPriority, $c)));
      }
      function bd(a) {
        function b(b2) {
          return ad(b2, a);
        }
        if (0 < Kc.length) {
          ad(Kc[0], a);
          for (var c = 1; c < Kc.length; c++) {
            var d = Kc[c];
            d.blockedOn === a && (d.blockedOn = null);
          }
        }
        null !== Lc && ad(Lc, a);
        null !== Mc && ad(Mc, a);
        null !== Nc && ad(Nc, a);
        Oc.forEach(b);
        Pc.forEach(b);
        for (c = 0; c < Qc.length; c++) d = Qc[c], d.blockedOn === a && (d.blockedOn = null);
        for (; 0 < Qc.length && (c = Qc[0], null === c.blockedOn); ) Vc(c), null === c.blockedOn && Qc.shift();
      }
      var cd = ua.ReactCurrentBatchConfig;
      var dd = true;
      function ed(a, b, c, d) {
        var e = C, f = cd.transition;
        cd.transition = null;
        try {
          C = 1, fd(a, b, c, d);
        } finally {
          C = e, cd.transition = f;
        }
      }
      function gd(a, b, c, d) {
        var e = C, f = cd.transition;
        cd.transition = null;
        try {
          C = 4, fd(a, b, c, d);
        } finally {
          C = e, cd.transition = f;
        }
      }
      function fd(a, b, c, d) {
        if (dd) {
          var e = Yc(a, b, c, d);
          if (null === e) hd(a, b, d, id, c), Sc(a, d);
          else if (Uc(e, a, b, c, d)) d.stopPropagation();
          else if (Sc(a, d), b & 4 && -1 < Rc.indexOf(a)) {
            for (; null !== e; ) {
              var f = Cb(e);
              null !== f && Ec(f);
              f = Yc(a, b, c, d);
              null === f && hd(a, b, d, id, c);
              if (f === e) break;
              e = f;
            }
            null !== e && d.stopPropagation();
          } else hd(a, b, d, null, c);
        }
      }
      var id = null;
      function Yc(a, b, c, d) {
        id = null;
        a = xb(d);
        a = Wc(a);
        if (null !== a) if (b = Vb(a), null === b) a = null;
        else if (c = b.tag, 13 === c) {
          a = Wb(b);
          if (null !== a) return a;
          a = null;
        } else if (3 === c) {
          if (b.stateNode.current.memoizedState.isDehydrated) return 3 === b.tag ? b.stateNode.containerInfo : null;
          a = null;
        } else b !== a && (a = null);
        id = a;
        return null;
      }
      function jd(a) {
        switch (a) {
          case "cancel":
          case "click":
          case "close":
          case "contextmenu":
          case "copy":
          case "cut":
          case "auxclick":
          case "dblclick":
          case "dragend":
          case "dragstart":
          case "drop":
          case "focusin":
          case "focusout":
          case "input":
          case "invalid":
          case "keydown":
          case "keypress":
          case "keyup":
          case "mousedown":
          case "mouseup":
          case "paste":
          case "pause":
          case "play":
          case "pointercancel":
          case "pointerdown":
          case "pointerup":
          case "ratechange":
          case "reset":
          case "resize":
          case "seeked":
          case "submit":
          case "touchcancel":
          case "touchend":
          case "touchstart":
          case "volumechange":
          case "change":
          case "selectionchange":
          case "textInput":
          case "compositionstart":
          case "compositionend":
          case "compositionupdate":
          case "beforeblur":
          case "afterblur":
          case "beforeinput":
          case "blur":
          case "fullscreenchange":
          case "focus":
          case "hashchange":
          case "popstate":
          case "select":
          case "selectstart":
            return 1;
          case "drag":
          case "dragenter":
          case "dragexit":
          case "dragleave":
          case "dragover":
          case "mousemove":
          case "mouseout":
          case "mouseover":
          case "pointermove":
          case "pointerout":
          case "pointerover":
          case "scroll":
          case "toggle":
          case "touchmove":
          case "wheel":
          case "mouseenter":
          case "mouseleave":
          case "pointerenter":
          case "pointerleave":
            return 4;
          case "message":
            switch (ec()) {
              case fc:
                return 1;
              case gc:
                return 4;
              case hc:
              case ic:
                return 16;
              case jc:
                return 536870912;
              default:
                return 16;
            }
          default:
            return 16;
        }
      }
      var kd = null;
      var ld = null;
      var md = null;
      function nd() {
        if (md) return md;
        var a, b = ld, c = b.length, d, e = "value" in kd ? kd.value : kd.textContent, f = e.length;
        for (a = 0; a < c && b[a] === e[a]; a++) ;
        var g = c - a;
        for (d = 1; d <= g && b[c - d] === e[f - d]; d++) ;
        return md = e.slice(a, 1 < d ? 1 - d : void 0);
      }
      function od(a) {
        var b = a.keyCode;
        "charCode" in a ? (a = a.charCode, 0 === a && 13 === b && (a = 13)) : a = b;
        10 === a && (a = 13);
        return 32 <= a || 13 === a ? a : 0;
      }
      function pd() {
        return true;
      }
      function qd() {
        return false;
      }
      function rd(a) {
        function b(b2, d, e, f, g) {
          this._reactName = b2;
          this._targetInst = e;
          this.type = d;
          this.nativeEvent = f;
          this.target = g;
          this.currentTarget = null;
          for (var c in a) a.hasOwnProperty(c) && (b2 = a[c], this[c] = b2 ? b2(f) : f[c]);
          this.isDefaultPrevented = (null != f.defaultPrevented ? f.defaultPrevented : false === f.returnValue) ? pd : qd;
          this.isPropagationStopped = qd;
          return this;
        }
        A(b.prototype, { preventDefault: function() {
          this.defaultPrevented = true;
          var a2 = this.nativeEvent;
          a2 && (a2.preventDefault ? a2.preventDefault() : "unknown" !== typeof a2.returnValue && (a2.returnValue = false), this.isDefaultPrevented = pd);
        }, stopPropagation: function() {
          var a2 = this.nativeEvent;
          a2 && (a2.stopPropagation ? a2.stopPropagation() : "unknown" !== typeof a2.cancelBubble && (a2.cancelBubble = true), this.isPropagationStopped = pd);
        }, persist: function() {
        }, isPersistent: pd });
        return b;
      }
      var sd = { eventPhase: 0, bubbles: 0, cancelable: 0, timeStamp: function(a) {
        return a.timeStamp || Date.now();
      }, defaultPrevented: 0, isTrusted: 0 };
      var td = rd(sd);
      var ud = A({}, sd, { view: 0, detail: 0 });
      var vd = rd(ud);
      var wd;
      var xd;
      var yd;
      var Ad = A({}, ud, { screenX: 0, screenY: 0, clientX: 0, clientY: 0, pageX: 0, pageY: 0, ctrlKey: 0, shiftKey: 0, altKey: 0, metaKey: 0, getModifierState: zd, button: 0, buttons: 0, relatedTarget: function(a) {
        return void 0 === a.relatedTarget ? a.fromElement === a.srcElement ? a.toElement : a.fromElement : a.relatedTarget;
      }, movementX: function(a) {
        if ("movementX" in a) return a.movementX;
        a !== yd && (yd && "mousemove" === a.type ? (wd = a.screenX - yd.screenX, xd = a.screenY - yd.screenY) : xd = wd = 0, yd = a);
        return wd;
      }, movementY: function(a) {
        return "movementY" in a ? a.movementY : xd;
      } });
      var Bd = rd(Ad);
      var Cd = A({}, Ad, { dataTransfer: 0 });
      var Dd = rd(Cd);
      var Ed = A({}, ud, { relatedTarget: 0 });
      var Fd = rd(Ed);
      var Gd = A({}, sd, { animationName: 0, elapsedTime: 0, pseudoElement: 0 });
      var Hd = rd(Gd);
      var Id = A({}, sd, { clipboardData: function(a) {
        return "clipboardData" in a ? a.clipboardData : window.clipboardData;
      } });
      var Jd = rd(Id);
      var Kd = A({}, sd, { data: 0 });
      var Ld = rd(Kd);
      var Md = {
        Esc: "Escape",
        Spacebar: " ",
        Left: "ArrowLeft",
        Up: "ArrowUp",
        Right: "ArrowRight",
        Down: "ArrowDown",
        Del: "Delete",
        Win: "OS",
        Menu: "ContextMenu",
        Apps: "ContextMenu",
        Scroll: "ScrollLock",
        MozPrintableKey: "Unidentified"
      };
      var Nd = {
        8: "Backspace",
        9: "Tab",
        12: "Clear",
        13: "Enter",
        16: "Shift",
        17: "Control",
        18: "Alt",
        19: "Pause",
        20: "CapsLock",
        27: "Escape",
        32: " ",
        33: "PageUp",
        34: "PageDown",
        35: "End",
        36: "Home",
        37: "ArrowLeft",
        38: "ArrowUp",
        39: "ArrowRight",
        40: "ArrowDown",
        45: "Insert",
        46: "Delete",
        112: "F1",
        113: "F2",
        114: "F3",
        115: "F4",
        116: "F5",
        117: "F6",
        118: "F7",
        119: "F8",
        120: "F9",
        121: "F10",
        122: "F11",
        123: "F12",
        144: "NumLock",
        145: "ScrollLock",
        224: "Meta"
      };
      var Od = { Alt: "altKey", Control: "ctrlKey", Meta: "metaKey", Shift: "shiftKey" };
      function Pd(a) {
        var b = this.nativeEvent;
        return b.getModifierState ? b.getModifierState(a) : (a = Od[a]) ? !!b[a] : false;
      }
      function zd() {
        return Pd;
      }
      var Qd = A({}, ud, { key: function(a) {
        if (a.key) {
          var b = Md[a.key] || a.key;
          if ("Unidentified" !== b) return b;
        }
        return "keypress" === a.type ? (a = od(a), 13 === a ? "Enter" : String.fromCharCode(a)) : "keydown" === a.type || "keyup" === a.type ? Nd[a.keyCode] || "Unidentified" : "";
      }, code: 0, location: 0, ctrlKey: 0, shiftKey: 0, altKey: 0, metaKey: 0, repeat: 0, locale: 0, getModifierState: zd, charCode: function(a) {
        return "keypress" === a.type ? od(a) : 0;
      }, keyCode: function(a) {
        return "keydown" === a.type || "keyup" === a.type ? a.keyCode : 0;
      }, which: function(a) {
        return "keypress" === a.type ? od(a) : "keydown" === a.type || "keyup" === a.type ? a.keyCode : 0;
      } });
      var Rd = rd(Qd);
      var Sd = A({}, Ad, { pointerId: 0, width: 0, height: 0, pressure: 0, tangentialPressure: 0, tiltX: 0, tiltY: 0, twist: 0, pointerType: 0, isPrimary: 0 });
      var Td = rd(Sd);
      var Ud = A({}, ud, { touches: 0, targetTouches: 0, changedTouches: 0, altKey: 0, metaKey: 0, ctrlKey: 0, shiftKey: 0, getModifierState: zd });
      var Vd = rd(Ud);
      var Wd = A({}, sd, { propertyName: 0, elapsedTime: 0, pseudoElement: 0 });
      var Xd = rd(Wd);
      var Yd = A({}, Ad, {
        deltaX: function(a) {
          return "deltaX" in a ? a.deltaX : "wheelDeltaX" in a ? -a.wheelDeltaX : 0;
        },
        deltaY: function(a) {
          return "deltaY" in a ? a.deltaY : "wheelDeltaY" in a ? -a.wheelDeltaY : "wheelDelta" in a ? -a.wheelDelta : 0;
        },
        deltaZ: 0,
        deltaMode: 0
      });
      var Zd = rd(Yd);
      var $d = [9, 13, 27, 32];
      var ae = ia && "CompositionEvent" in window;
      var be = null;
      ia && "documentMode" in document && (be = document.documentMode);
      var ce = ia && "TextEvent" in window && !be;
      var de = ia && (!ae || be && 8 < be && 11 >= be);
      var ee = String.fromCharCode(32);
      var fe = false;
      function ge(a, b) {
        switch (a) {
          case "keyup":
            return -1 !== $d.indexOf(b.keyCode);
          case "keydown":
            return 229 !== b.keyCode;
          case "keypress":
          case "mousedown":
          case "focusout":
            return true;
          default:
            return false;
        }
      }
      function he(a) {
        a = a.detail;
        return "object" === typeof a && "data" in a ? a.data : null;
      }
      var ie = false;
      function je(a, b) {
        switch (a) {
          case "compositionend":
            return he(b);
          case "keypress":
            if (32 !== b.which) return null;
            fe = true;
            return ee;
          case "textInput":
            return a = b.data, a === ee && fe ? null : a;
          default:
            return null;
        }
      }
      function ke(a, b) {
        if (ie) return "compositionend" === a || !ae && ge(a, b) ? (a = nd(), md = ld = kd = null, ie = false, a) : null;
        switch (a) {
          case "paste":
            return null;
          case "keypress":
            if (!(b.ctrlKey || b.altKey || b.metaKey) || b.ctrlKey && b.altKey) {
              if (b.char && 1 < b.char.length) return b.char;
              if (b.which) return String.fromCharCode(b.which);
            }
            return null;
          case "compositionend":
            return de && "ko" !== b.locale ? null : b.data;
          default:
            return null;
        }
      }
      var le = { color: true, date: true, datetime: true, "datetime-local": true, email: true, month: true, number: true, password: true, range: true, search: true, tel: true, text: true, time: true, url: true, week: true };
      function me(a) {
        var b = a && a.nodeName && a.nodeName.toLowerCase();
        return "input" === b ? !!le[a.type] : "textarea" === b ? true : false;
      }
      function ne(a, b, c, d) {
        Eb(d);
        b = oe(b, "onChange");
        0 < b.length && (c = new td("onChange", "change", null, c, d), a.push({ event: c, listeners: b }));
      }
      var pe = null;
      var qe = null;
      function re(a) {
        se(a, 0);
      }
      function te(a) {
        var b = ue(a);
        if (Wa(b)) return a;
      }
      function ve(a, b) {
        if ("change" === a) return b;
      }
      var we = false;
      if (ia) {
        if (ia) {
          ye = "oninput" in document;
          if (!ye) {
            ze = document.createElement("div");
            ze.setAttribute("oninput", "return;");
            ye = "function" === typeof ze.oninput;
          }
          xe = ye;
        } else xe = false;
        we = xe && (!document.documentMode || 9 < document.documentMode);
      }
      var xe;
      var ye;
      var ze;
      function Ae() {
        pe && (pe.detachEvent("onpropertychange", Be), qe = pe = null);
      }
      function Be(a) {
        if ("value" === a.propertyName && te(qe)) {
          var b = [];
          ne(b, qe, a, xb(a));
          Jb(re, b);
        }
      }
      function Ce(a, b, c) {
        "focusin" === a ? (Ae(), pe = b, qe = c, pe.attachEvent("onpropertychange", Be)) : "focusout" === a && Ae();
      }
      function De(a) {
        if ("selectionchange" === a || "keyup" === a || "keydown" === a) return te(qe);
      }
      function Ee(a, b) {
        if ("click" === a) return te(b);
      }
      function Fe(a, b) {
        if ("input" === a || "change" === a) return te(b);
      }
      function Ge(a, b) {
        return a === b && (0 !== a || 1 / a === 1 / b) || a !== a && b !== b;
      }
      var He = "function" === typeof Object.is ? Object.is : Ge;
      function Ie(a, b) {
        if (He(a, b)) return true;
        if ("object" !== typeof a || null === a || "object" !== typeof b || null === b) return false;
        var c = Object.keys(a), d = Object.keys(b);
        if (c.length !== d.length) return false;
        for (d = 0; d < c.length; d++) {
          var e = c[d];
          if (!ja.call(b, e) || !He(a[e], b[e])) return false;
        }
        return true;
      }
      function Je(a) {
        for (; a && a.firstChild; ) a = a.firstChild;
        return a;
      }
      function Ke(a, b) {
        var c = Je(a);
        a = 0;
        for (var d; c; ) {
          if (3 === c.nodeType) {
            d = a + c.textContent.length;
            if (a <= b && d >= b) return { node: c, offset: b - a };
            a = d;
          }
          a: {
            for (; c; ) {
              if (c.nextSibling) {
                c = c.nextSibling;
                break a;
              }
              c = c.parentNode;
            }
            c = void 0;
          }
          c = Je(c);
        }
      }
      function Le(a, b) {
        return a && b ? a === b ? true : a && 3 === a.nodeType ? false : b && 3 === b.nodeType ? Le(a, b.parentNode) : "contains" in a ? a.contains(b) : a.compareDocumentPosition ? !!(a.compareDocumentPosition(b) & 16) : false : false;
      }
      function Me() {
        for (var a = window, b = Xa(); b instanceof a.HTMLIFrameElement; ) {
          try {
            var c = "string" === typeof b.contentWindow.location.href;
          } catch (d) {
            c = false;
          }
          if (c) a = b.contentWindow;
          else break;
          b = Xa(a.document);
        }
        return b;
      }
      function Ne(a) {
        var b = a && a.nodeName && a.nodeName.toLowerCase();
        return b && ("input" === b && ("text" === a.type || "search" === a.type || "tel" === a.type || "url" === a.type || "password" === a.type) || "textarea" === b || "true" === a.contentEditable);
      }
      function Oe(a) {
        var b = Me(), c = a.focusedElem, d = a.selectionRange;
        if (b !== c && c && c.ownerDocument && Le(c.ownerDocument.documentElement, c)) {
          if (null !== d && Ne(c)) {
            if (b = d.start, a = d.end, void 0 === a && (a = b), "selectionStart" in c) c.selectionStart = b, c.selectionEnd = Math.min(a, c.value.length);
            else if (a = (b = c.ownerDocument || document) && b.defaultView || window, a.getSelection) {
              a = a.getSelection();
              var e = c.textContent.length, f = Math.min(d.start, e);
              d = void 0 === d.end ? f : Math.min(d.end, e);
              !a.extend && f > d && (e = d, d = f, f = e);
              e = Ke(c, f);
              var g = Ke(
                c,
                d
              );
              e && g && (1 !== a.rangeCount || a.anchorNode !== e.node || a.anchorOffset !== e.offset || a.focusNode !== g.node || a.focusOffset !== g.offset) && (b = b.createRange(), b.setStart(e.node, e.offset), a.removeAllRanges(), f > d ? (a.addRange(b), a.extend(g.node, g.offset)) : (b.setEnd(g.node, g.offset), a.addRange(b)));
            }
          }
          b = [];
          for (a = c; a = a.parentNode; ) 1 === a.nodeType && b.push({ element: a, left: a.scrollLeft, top: a.scrollTop });
          "function" === typeof c.focus && c.focus();
          for (c = 0; c < b.length; c++) a = b[c], a.element.scrollLeft = a.left, a.element.scrollTop = a.top;
        }
      }
      var Pe = ia && "documentMode" in document && 11 >= document.documentMode;
      var Qe = null;
      var Re = null;
      var Se = null;
      var Te = false;
      function Ue(a, b, c) {
        var d = c.window === c ? c.document : 9 === c.nodeType ? c : c.ownerDocument;
        Te || null == Qe || Qe !== Xa(d) || (d = Qe, "selectionStart" in d && Ne(d) ? d = { start: d.selectionStart, end: d.selectionEnd } : (d = (d.ownerDocument && d.ownerDocument.defaultView || window).getSelection(), d = { anchorNode: d.anchorNode, anchorOffset: d.anchorOffset, focusNode: d.focusNode, focusOffset: d.focusOffset }), Se && Ie(Se, d) || (Se = d, d = oe(Re, "onSelect"), 0 < d.length && (b = new td("onSelect", "select", null, b, c), a.push({ event: b, listeners: d }), b.target = Qe)));
      }
      function Ve(a, b) {
        var c = {};
        c[a.toLowerCase()] = b.toLowerCase();
        c["Webkit" + a] = "webkit" + b;
        c["Moz" + a] = "moz" + b;
        return c;
      }
      var We = { animationend: Ve("Animation", "AnimationEnd"), animationiteration: Ve("Animation", "AnimationIteration"), animationstart: Ve("Animation", "AnimationStart"), transitionend: Ve("Transition", "TransitionEnd") };
      var Xe = {};
      var Ye = {};
      ia && (Ye = document.createElement("div").style, "AnimationEvent" in window || (delete We.animationend.animation, delete We.animationiteration.animation, delete We.animationstart.animation), "TransitionEvent" in window || delete We.transitionend.transition);
      function Ze(a) {
        if (Xe[a]) return Xe[a];
        if (!We[a]) return a;
        var b = We[a], c;
        for (c in b) if (b.hasOwnProperty(c) && c in Ye) return Xe[a] = b[c];
        return a;
      }
      var $e = Ze("animationend");
      var af = Ze("animationiteration");
      var bf = Ze("animationstart");
      var cf = Ze("transitionend");
      var df = /* @__PURE__ */ new Map();
      var ef = "abort auxClick cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel".split(" ");
      function ff(a, b) {
        df.set(a, b);
        fa(b, [a]);
      }
      for (gf = 0; gf < ef.length; gf++) {
        hf = ef[gf], jf = hf.toLowerCase(), kf = hf[0].toUpperCase() + hf.slice(1);
        ff(jf, "on" + kf);
      }
      var hf;
      var jf;
      var kf;
      var gf;
      ff($e, "onAnimationEnd");
      ff(af, "onAnimationIteration");
      ff(bf, "onAnimationStart");
      ff("dblclick", "onDoubleClick");
      ff("focusin", "onFocus");
      ff("focusout", "onBlur");
      ff(cf, "onTransitionEnd");
      ha("onMouseEnter", ["mouseout", "mouseover"]);
      ha("onMouseLeave", ["mouseout", "mouseover"]);
      ha("onPointerEnter", ["pointerout", "pointerover"]);
      ha("onPointerLeave", ["pointerout", "pointerover"]);
      fa("onChange", "change click focusin focusout input keydown keyup selectionchange".split(" "));
      fa("onSelect", "focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange".split(" "));
      fa("onBeforeInput", ["compositionend", "keypress", "textInput", "paste"]);
      fa("onCompositionEnd", "compositionend focusout keydown keypress keyup mousedown".split(" "));
      fa("onCompositionStart", "compositionstart focusout keydown keypress keyup mousedown".split(" "));
      fa("onCompositionUpdate", "compositionupdate focusout keydown keypress keyup mousedown".split(" "));
      var lf = "abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting".split(" ");
      var mf = new Set("cancel close invalid load scroll toggle".split(" ").concat(lf));
      function nf(a, b, c) {
        var d = a.type || "unknown-event";
        a.currentTarget = c;
        Ub(d, b, void 0, a);
        a.currentTarget = null;
      }
      function se(a, b) {
        b = 0 !== (b & 4);
        for (var c = 0; c < a.length; c++) {
          var d = a[c], e = d.event;
          d = d.listeners;
          a: {
            var f = void 0;
            if (b) for (var g = d.length - 1; 0 <= g; g--) {
              var h = d[g], k = h.instance, l = h.currentTarget;
              h = h.listener;
              if (k !== f && e.isPropagationStopped()) break a;
              nf(e, h, l);
              f = k;
            }
            else for (g = 0; g < d.length; g++) {
              h = d[g];
              k = h.instance;
              l = h.currentTarget;
              h = h.listener;
              if (k !== f && e.isPropagationStopped()) break a;
              nf(e, h, l);
              f = k;
            }
          }
        }
        if (Qb) throw a = Rb, Qb = false, Rb = null, a;
      }
      function D(a, b) {
        var c = b[of];
        void 0 === c && (c = b[of] = /* @__PURE__ */ new Set());
        var d = a + "__bubble";
        c.has(d) || (pf(b, a, 2, false), c.add(d));
      }
      function qf(a, b, c) {
        var d = 0;
        b && (d |= 4);
        pf(c, a, d, b);
      }
      var rf = "_reactListening" + Math.random().toString(36).slice(2);
      function sf(a) {
        if (!a[rf]) {
          a[rf] = true;
          da.forEach(function(b2) {
            "selectionchange" !== b2 && (mf.has(b2) || qf(b2, false, a), qf(b2, true, a));
          });
          var b = 9 === a.nodeType ? a : a.ownerDocument;
          null === b || b[rf] || (b[rf] = true, qf("selectionchange", false, b));
        }
      }
      function pf(a, b, c, d) {
        switch (jd(b)) {
          case 1:
            var e = ed;
            break;
          case 4:
            e = gd;
            break;
          default:
            e = fd;
        }
        c = e.bind(null, b, c, a);
        e = void 0;
        !Lb || "touchstart" !== b && "touchmove" !== b && "wheel" !== b || (e = true);
        d ? void 0 !== e ? a.addEventListener(b, c, { capture: true, passive: e }) : a.addEventListener(b, c, true) : void 0 !== e ? a.addEventListener(b, c, { passive: e }) : a.addEventListener(b, c, false);
      }
      function hd(a, b, c, d, e) {
        var f = d;
        if (0 === (b & 1) && 0 === (b & 2) && null !== d) a: for (; ; ) {
          if (null === d) return;
          var g = d.tag;
          if (3 === g || 4 === g) {
            var h = d.stateNode.containerInfo;
            if (h === e || 8 === h.nodeType && h.parentNode === e) break;
            if (4 === g) for (g = d.return; null !== g; ) {
              var k = g.tag;
              if (3 === k || 4 === k) {
                if (k = g.stateNode.containerInfo, k === e || 8 === k.nodeType && k.parentNode === e) return;
              }
              g = g.return;
            }
            for (; null !== h; ) {
              g = Wc(h);
              if (null === g) return;
              k = g.tag;
              if (5 === k || 6 === k) {
                d = f = g;
                continue a;
              }
              h = h.parentNode;
            }
          }
          d = d.return;
        }
        Jb(function() {
          var d2 = f, e2 = xb(c), g2 = [];
          a: {
            var h2 = df.get(a);
            if (void 0 !== h2) {
              var k2 = td, n = a;
              switch (a) {
                case "keypress":
                  if (0 === od(c)) break a;
                case "keydown":
                case "keyup":
                  k2 = Rd;
                  break;
                case "focusin":
                  n = "focus";
                  k2 = Fd;
                  break;
                case "focusout":
                  n = "blur";
                  k2 = Fd;
                  break;
                case "beforeblur":
                case "afterblur":
                  k2 = Fd;
                  break;
                case "click":
                  if (2 === c.button) break a;
                case "auxclick":
                case "dblclick":
                case "mousedown":
                case "mousemove":
                case "mouseup":
                case "mouseout":
                case "mouseover":
                case "contextmenu":
                  k2 = Bd;
                  break;
                case "drag":
                case "dragend":
                case "dragenter":
                case "dragexit":
                case "dragleave":
                case "dragover":
                case "dragstart":
                case "drop":
                  k2 = Dd;
                  break;
                case "touchcancel":
                case "touchend":
                case "touchmove":
                case "touchstart":
                  k2 = Vd;
                  break;
                case $e:
                case af:
                case bf:
                  k2 = Hd;
                  break;
                case cf:
                  k2 = Xd;
                  break;
                case "scroll":
                  k2 = vd;
                  break;
                case "wheel":
                  k2 = Zd;
                  break;
                case "copy":
                case "cut":
                case "paste":
                  k2 = Jd;
                  break;
                case "gotpointercapture":
                case "lostpointercapture":
                case "pointercancel":
                case "pointerdown":
                case "pointermove":
                case "pointerout":
                case "pointerover":
                case "pointerup":
                  k2 = Td;
              }
              var t = 0 !== (b & 4), J = !t && "scroll" === a, x = t ? null !== h2 ? h2 + "Capture" : null : h2;
              t = [];
              for (var w = d2, u; null !== w; ) {
                u = w;
                var F = u.stateNode;
                5 === u.tag && null !== F && (u = F, null !== x && (F = Kb(w, x), null != F && t.push(tf(w, F, u))));
                if (J) break;
                w = w.return;
              }
              0 < t.length && (h2 = new k2(h2, n, null, c, e2), g2.push({ event: h2, listeners: t }));
            }
          }
          if (0 === (b & 7)) {
            a: {
              h2 = "mouseover" === a || "pointerover" === a;
              k2 = "mouseout" === a || "pointerout" === a;
              if (h2 && c !== wb && (n = c.relatedTarget || c.fromElement) && (Wc(n) || n[uf])) break a;
              if (k2 || h2) {
                h2 = e2.window === e2 ? e2 : (h2 = e2.ownerDocument) ? h2.defaultView || h2.parentWindow : window;
                if (k2) {
                  if (n = c.relatedTarget || c.toElement, k2 = d2, n = n ? Wc(n) : null, null !== n && (J = Vb(n), n !== J || 5 !== n.tag && 6 !== n.tag)) n = null;
                } else k2 = null, n = d2;
                if (k2 !== n) {
                  t = Bd;
                  F = "onMouseLeave";
                  x = "onMouseEnter";
                  w = "mouse";
                  if ("pointerout" === a || "pointerover" === a) t = Td, F = "onPointerLeave", x = "onPointerEnter", w = "pointer";
                  J = null == k2 ? h2 : ue(k2);
                  u = null == n ? h2 : ue(n);
                  h2 = new t(F, w + "leave", k2, c, e2);
                  h2.target = J;
                  h2.relatedTarget = u;
                  F = null;
                  Wc(e2) === d2 && (t = new t(x, w + "enter", n, c, e2), t.target = u, t.relatedTarget = J, F = t);
                  J = F;
                  if (k2 && n) b: {
                    t = k2;
                    x = n;
                    w = 0;
                    for (u = t; u; u = vf(u)) w++;
                    u = 0;
                    for (F = x; F; F = vf(F)) u++;
                    for (; 0 < w - u; ) t = vf(t), w--;
                    for (; 0 < u - w; ) x = vf(x), u--;
                    for (; w--; ) {
                      if (t === x || null !== x && t === x.alternate) break b;
                      t = vf(t);
                      x = vf(x);
                    }
                    t = null;
                  }
                  else t = null;
                  null !== k2 && wf(g2, h2, k2, t, false);
                  null !== n && null !== J && wf(g2, J, n, t, true);
                }
              }
            }
            a: {
              h2 = d2 ? ue(d2) : window;
              k2 = h2.nodeName && h2.nodeName.toLowerCase();
              if ("select" === k2 || "input" === k2 && "file" === h2.type) var na = ve;
              else if (me(h2)) if (we) na = Fe;
              else {
                na = De;
                var xa = Ce;
              }
              else (k2 = h2.nodeName) && "input" === k2.toLowerCase() && ("checkbox" === h2.type || "radio" === h2.type) && (na = Ee);
              if (na && (na = na(a, d2))) {
                ne(g2, na, c, e2);
                break a;
              }
              xa && xa(a, h2, d2);
              "focusout" === a && (xa = h2._wrapperState) && xa.controlled && "number" === h2.type && cb(h2, "number", h2.value);
            }
            xa = d2 ? ue(d2) : window;
            switch (a) {
              case "focusin":
                if (me(xa) || "true" === xa.contentEditable) Qe = xa, Re = d2, Se = null;
                break;
              case "focusout":
                Se = Re = Qe = null;
                break;
              case "mousedown":
                Te = true;
                break;
              case "contextmenu":
              case "mouseup":
              case "dragend":
                Te = false;
                Ue(g2, c, e2);
                break;
              case "selectionchange":
                if (Pe) break;
              case "keydown":
              case "keyup":
                Ue(g2, c, e2);
            }
            var $a;
            if (ae) b: {
              switch (a) {
                case "compositionstart":
                  var ba = "onCompositionStart";
                  break b;
                case "compositionend":
                  ba = "onCompositionEnd";
                  break b;
                case "compositionupdate":
                  ba = "onCompositionUpdate";
                  break b;
              }
              ba = void 0;
            }
            else ie ? ge(a, c) && (ba = "onCompositionEnd") : "keydown" === a && 229 === c.keyCode && (ba = "onCompositionStart");
            ba && (de && "ko" !== c.locale && (ie || "onCompositionStart" !== ba ? "onCompositionEnd" === ba && ie && ($a = nd()) : (kd = e2, ld = "value" in kd ? kd.value : kd.textContent, ie = true)), xa = oe(d2, ba), 0 < xa.length && (ba = new Ld(ba, a, null, c, e2), g2.push({ event: ba, listeners: xa }), $a ? ba.data = $a : ($a = he(c), null !== $a && (ba.data = $a))));
            if ($a = ce ? je(a, c) : ke(a, c)) d2 = oe(d2, "onBeforeInput"), 0 < d2.length && (e2 = new Ld("onBeforeInput", "beforeinput", null, c, e2), g2.push({ event: e2, listeners: d2 }), e2.data = $a);
          }
          se(g2, b);
        });
      }
      function tf(a, b, c) {
        return { instance: a, listener: b, currentTarget: c };
      }
      function oe(a, b) {
        for (var c = b + "Capture", d = []; null !== a; ) {
          var e = a, f = e.stateNode;
          5 === e.tag && null !== f && (e = f, f = Kb(a, c), null != f && d.unshift(tf(a, f, e)), f = Kb(a, b), null != f && d.push(tf(a, f, e)));
          a = a.return;
        }
        return d;
      }
      function vf(a) {
        if (null === a) return null;
        do
          a = a.return;
        while (a && 5 !== a.tag);
        return a ? a : null;
      }
      function wf(a, b, c, d, e) {
        for (var f = b._reactName, g = []; null !== c && c !== d; ) {
          var h = c, k = h.alternate, l = h.stateNode;
          if (null !== k && k === d) break;
          5 === h.tag && null !== l && (h = l, e ? (k = Kb(c, f), null != k && g.unshift(tf(c, k, h))) : e || (k = Kb(c, f), null != k && g.push(tf(c, k, h))));
          c = c.return;
        }
        0 !== g.length && a.push({ event: b, listeners: g });
      }
      var xf = /\r\n?/g;
      var yf = /\u0000|\uFFFD/g;
      function zf(a) {
        return ("string" === typeof a ? a : "" + a).replace(xf, "\n").replace(yf, "");
      }
      function Af(a, b, c) {
        b = zf(b);
        if (zf(a) !== b && c) throw Error(p(425));
      }
      function Bf() {
      }
      var Cf = null;
      var Df = null;
      function Ef(a, b) {
        return "textarea" === a || "noscript" === a || "string" === typeof b.children || "number" === typeof b.children || "object" === typeof b.dangerouslySetInnerHTML && null !== b.dangerouslySetInnerHTML && null != b.dangerouslySetInnerHTML.__html;
      }
      var Ff = "function" === typeof setTimeout ? setTimeout : void 0;
      var Gf = "function" === typeof clearTimeout ? clearTimeout : void 0;
      var Hf = "function" === typeof Promise ? Promise : void 0;
      var Jf = "function" === typeof queueMicrotask ? queueMicrotask : "undefined" !== typeof Hf ? function(a) {
        return Hf.resolve(null).then(a).catch(If);
      } : Ff;
      function If(a) {
        setTimeout(function() {
          throw a;
        });
      }
      function Kf(a, b) {
        var c = b, d = 0;
        do {
          var e = c.nextSibling;
          a.removeChild(c);
          if (e && 8 === e.nodeType) if (c = e.data, "/$" === c) {
            if (0 === d) {
              a.removeChild(e);
              bd(b);
              return;
            }
            d--;
          } else "$" !== c && "$?" !== c && "$!" !== c || d++;
          c = e;
        } while (c);
        bd(b);
      }
      function Lf(a) {
        for (; null != a; a = a.nextSibling) {
          var b = a.nodeType;
          if (1 === b || 3 === b) break;
          if (8 === b) {
            b = a.data;
            if ("$" === b || "$!" === b || "$?" === b) break;
            if ("/$" === b) return null;
          }
        }
        return a;
      }
      function Mf(a) {
        a = a.previousSibling;
        for (var b = 0; a; ) {
          if (8 === a.nodeType) {
            var c = a.data;
            if ("$" === c || "$!" === c || "$?" === c) {
              if (0 === b) return a;
              b--;
            } else "/$" === c && b++;
          }
          a = a.previousSibling;
        }
        return null;
      }
      var Nf = Math.random().toString(36).slice(2);
      var Of = "__reactFiber$" + Nf;
      var Pf = "__reactProps$" + Nf;
      var uf = "__reactContainer$" + Nf;
      var of = "__reactEvents$" + Nf;
      var Qf = "__reactListeners$" + Nf;
      var Rf = "__reactHandles$" + Nf;
      function Wc(a) {
        var b = a[Of];
        if (b) return b;
        for (var c = a.parentNode; c; ) {
          if (b = c[uf] || c[Of]) {
            c = b.alternate;
            if (null !== b.child || null !== c && null !== c.child) for (a = Mf(a); null !== a; ) {
              if (c = a[Of]) return c;
              a = Mf(a);
            }
            return b;
          }
          a = c;
          c = a.parentNode;
        }
        return null;
      }
      function Cb(a) {
        a = a[Of] || a[uf];
        return !a || 5 !== a.tag && 6 !== a.tag && 13 !== a.tag && 3 !== a.tag ? null : a;
      }
      function ue(a) {
        if (5 === a.tag || 6 === a.tag) return a.stateNode;
        throw Error(p(33));
      }
      function Db(a) {
        return a[Pf] || null;
      }
      var Sf = [];
      var Tf = -1;
      function Uf(a) {
        return { current: a };
      }
      function E(a) {
        0 > Tf || (a.current = Sf[Tf], Sf[Tf] = null, Tf--);
      }
      function G(a, b) {
        Tf++;
        Sf[Tf] = a.current;
        a.current = b;
      }
      var Vf = {};
      var H = Uf(Vf);
      var Wf = Uf(false);
      var Xf = Vf;
      function Yf(a, b) {
        var c = a.type.contextTypes;
        if (!c) return Vf;
        var d = a.stateNode;
        if (d && d.__reactInternalMemoizedUnmaskedChildContext === b) return d.__reactInternalMemoizedMaskedChildContext;
        var e = {}, f;
        for (f in c) e[f] = b[f];
        d && (a = a.stateNode, a.__reactInternalMemoizedUnmaskedChildContext = b, a.__reactInternalMemoizedMaskedChildContext = e);
        return e;
      }
      function Zf(a) {
        a = a.childContextTypes;
        return null !== a && void 0 !== a;
      }
      function $f() {
        E(Wf);
        E(H);
      }
      function ag(a, b, c) {
        if (H.current !== Vf) throw Error(p(168));
        G(H, b);
        G(Wf, c);
      }
      function bg(a, b, c) {
        var d = a.stateNode;
        b = b.childContextTypes;
        if ("function" !== typeof d.getChildContext) return c;
        d = d.getChildContext();
        for (var e in d) if (!(e in b)) throw Error(p(108, Ra(a) || "Unknown", e));
        return A({}, c, d);
      }
      function cg(a) {
        a = (a = a.stateNode) && a.__reactInternalMemoizedMergedChildContext || Vf;
        Xf = H.current;
        G(H, a);
        G(Wf, Wf.current);
        return true;
      }
      function dg(a, b, c) {
        var d = a.stateNode;
        if (!d) throw Error(p(169));
        c ? (a = bg(a, b, Xf), d.__reactInternalMemoizedMergedChildContext = a, E(Wf), E(H), G(H, a)) : E(Wf);
        G(Wf, c);
      }
      var eg = null;
      var fg = false;
      var gg = false;
      function hg(a) {
        null === eg ? eg = [a] : eg.push(a);
      }
      function ig(a) {
        fg = true;
        hg(a);
      }
      function jg() {
        if (!gg && null !== eg) {
          gg = true;
          var a = 0, b = C;
          try {
            var c = eg;
            for (C = 1; a < c.length; a++) {
              var d = c[a];
              do
                d = d(true);
              while (null !== d);
            }
            eg = null;
            fg = false;
          } catch (e) {
            throw null !== eg && (eg = eg.slice(a + 1)), ac(fc, jg), e;
          } finally {
            C = b, gg = false;
          }
        }
        return null;
      }
      var kg = [];
      var lg = 0;
      var mg = null;
      var ng = 0;
      var og = [];
      var pg = 0;
      var qg = null;
      var rg = 1;
      var sg = "";
      function tg(a, b) {
        kg[lg++] = ng;
        kg[lg++] = mg;
        mg = a;
        ng = b;
      }
      function ug(a, b, c) {
        og[pg++] = rg;
        og[pg++] = sg;
        og[pg++] = qg;
        qg = a;
        var d = rg;
        a = sg;
        var e = 32 - oc(d) - 1;
        d &= ~(1 << e);
        c += 1;
        var f = 32 - oc(b) + e;
        if (30 < f) {
          var g = e - e % 5;
          f = (d & (1 << g) - 1).toString(32);
          d >>= g;
          e -= g;
          rg = 1 << 32 - oc(b) + e | c << e | d;
          sg = f + a;
        } else rg = 1 << f | c << e | d, sg = a;
      }
      function vg(a) {
        null !== a.return && (tg(a, 1), ug(a, 1, 0));
      }
      function wg(a) {
        for (; a === mg; ) mg = kg[--lg], kg[lg] = null, ng = kg[--lg], kg[lg] = null;
        for (; a === qg; ) qg = og[--pg], og[pg] = null, sg = og[--pg], og[pg] = null, rg = og[--pg], og[pg] = null;
      }
      var xg = null;
      var yg = null;
      var I = false;
      var zg = null;
      function Ag(a, b) {
        var c = Bg(5, null, null, 0);
        c.elementType = "DELETED";
        c.stateNode = b;
        c.return = a;
        b = a.deletions;
        null === b ? (a.deletions = [c], a.flags |= 16) : b.push(c);
      }
      function Cg(a, b) {
        switch (a.tag) {
          case 5:
            var c = a.type;
            b = 1 !== b.nodeType || c.toLowerCase() !== b.nodeName.toLowerCase() ? null : b;
            return null !== b ? (a.stateNode = b, xg = a, yg = Lf(b.firstChild), true) : false;
          case 6:
            return b = "" === a.pendingProps || 3 !== b.nodeType ? null : b, null !== b ? (a.stateNode = b, xg = a, yg = null, true) : false;
          case 13:
            return b = 8 !== b.nodeType ? null : b, null !== b ? (c = null !== qg ? { id: rg, overflow: sg } : null, a.memoizedState = { dehydrated: b, treeContext: c, retryLane: 1073741824 }, c = Bg(18, null, null, 0), c.stateNode = b, c.return = a, a.child = c, xg = a, yg = null, true) : false;
          default:
            return false;
        }
      }
      function Dg(a) {
        return 0 !== (a.mode & 1) && 0 === (a.flags & 128);
      }
      function Eg(a) {
        if (I) {
          var b = yg;
          if (b) {
            var c = b;
            if (!Cg(a, b)) {
              if (Dg(a)) throw Error(p(418));
              b = Lf(c.nextSibling);
              var d = xg;
              b && Cg(a, b) ? Ag(d, c) : (a.flags = a.flags & -4097 | 2, I = false, xg = a);
            }
          } else {
            if (Dg(a)) throw Error(p(418));
            a.flags = a.flags & -4097 | 2;
            I = false;
            xg = a;
          }
        }
      }
      function Fg(a) {
        for (a = a.return; null !== a && 5 !== a.tag && 3 !== a.tag && 13 !== a.tag; ) a = a.return;
        xg = a;
      }
      function Gg(a) {
        if (a !== xg) return false;
        if (!I) return Fg(a), I = true, false;
        var b;
        (b = 3 !== a.tag) && !(b = 5 !== a.tag) && (b = a.type, b = "head" !== b && "body" !== b && !Ef(a.type, a.memoizedProps));
        if (b && (b = yg)) {
          if (Dg(a)) throw Hg(), Error(p(418));
          for (; b; ) Ag(a, b), b = Lf(b.nextSibling);
        }
        Fg(a);
        if (13 === a.tag) {
          a = a.memoizedState;
          a = null !== a ? a.dehydrated : null;
          if (!a) throw Error(p(317));
          a: {
            a = a.nextSibling;
            for (b = 0; a; ) {
              if (8 === a.nodeType) {
                var c = a.data;
                if ("/$" === c) {
                  if (0 === b) {
                    yg = Lf(a.nextSibling);
                    break a;
                  }
                  b--;
                } else "$" !== c && "$!" !== c && "$?" !== c || b++;
              }
              a = a.nextSibling;
            }
            yg = null;
          }
        } else yg = xg ? Lf(a.stateNode.nextSibling) : null;
        return true;
      }
      function Hg() {
        for (var a = yg; a; ) a = Lf(a.nextSibling);
      }
      function Ig() {
        yg = xg = null;
        I = false;
      }
      function Jg(a) {
        null === zg ? zg = [a] : zg.push(a);
      }
      var Kg = ua.ReactCurrentBatchConfig;
      function Lg(a, b, c) {
        a = c.ref;
        if (null !== a && "function" !== typeof a && "object" !== typeof a) {
          if (c._owner) {
            c = c._owner;
            if (c) {
              if (1 !== c.tag) throw Error(p(309));
              var d = c.stateNode;
            }
            if (!d) throw Error(p(147, a));
            var e = d, f = "" + a;
            if (null !== b && null !== b.ref && "function" === typeof b.ref && b.ref._stringRef === f) return b.ref;
            b = function(a2) {
              var b2 = e.refs;
              null === a2 ? delete b2[f] : b2[f] = a2;
            };
            b._stringRef = f;
            return b;
          }
          if ("string" !== typeof a) throw Error(p(284));
          if (!c._owner) throw Error(p(290, a));
        }
        return a;
      }
      function Mg(a, b) {
        a = Object.prototype.toString.call(b);
        throw Error(p(31, "[object Object]" === a ? "object with keys {" + Object.keys(b).join(", ") + "}" : a));
      }
      function Ng(a) {
        var b = a._init;
        return b(a._payload);
      }
      function Og(a) {
        function b(b2, c2) {
          if (a) {
            var d2 = b2.deletions;
            null === d2 ? (b2.deletions = [c2], b2.flags |= 16) : d2.push(c2);
          }
        }
        function c(c2, d2) {
          if (!a) return null;
          for (; null !== d2; ) b(c2, d2), d2 = d2.sibling;
          return null;
        }
        function d(a2, b2) {
          for (a2 = /* @__PURE__ */ new Map(); null !== b2; ) null !== b2.key ? a2.set(b2.key, b2) : a2.set(b2.index, b2), b2 = b2.sibling;
          return a2;
        }
        function e(a2, b2) {
          a2 = Pg(a2, b2);
          a2.index = 0;
          a2.sibling = null;
          return a2;
        }
        function f(b2, c2, d2) {
          b2.index = d2;
          if (!a) return b2.flags |= 1048576, c2;
          d2 = b2.alternate;
          if (null !== d2) return d2 = d2.index, d2 < c2 ? (b2.flags |= 2, c2) : d2;
          b2.flags |= 2;
          return c2;
        }
        function g(b2) {
          a && null === b2.alternate && (b2.flags |= 2);
          return b2;
        }
        function h(a2, b2, c2, d2) {
          if (null === b2 || 6 !== b2.tag) return b2 = Qg(c2, a2.mode, d2), b2.return = a2, b2;
          b2 = e(b2, c2);
          b2.return = a2;
          return b2;
        }
        function k(a2, b2, c2, d2) {
          var f2 = c2.type;
          if (f2 === ya) return m(a2, b2, c2.props.children, d2, c2.key);
          if (null !== b2 && (b2.elementType === f2 || "object" === typeof f2 && null !== f2 && f2.$$typeof === Ha && Ng(f2) === b2.type)) return d2 = e(b2, c2.props), d2.ref = Lg(a2, b2, c2), d2.return = a2, d2;
          d2 = Rg(c2.type, c2.key, c2.props, null, a2.mode, d2);
          d2.ref = Lg(a2, b2, c2);
          d2.return = a2;
          return d2;
        }
        function l(a2, b2, c2, d2) {
          if (null === b2 || 4 !== b2.tag || b2.stateNode.containerInfo !== c2.containerInfo || b2.stateNode.implementation !== c2.implementation) return b2 = Sg(c2, a2.mode, d2), b2.return = a2, b2;
          b2 = e(b2, c2.children || []);
          b2.return = a2;
          return b2;
        }
        function m(a2, b2, c2, d2, f2) {
          if (null === b2 || 7 !== b2.tag) return b2 = Tg(c2, a2.mode, d2, f2), b2.return = a2, b2;
          b2 = e(b2, c2);
          b2.return = a2;
          return b2;
        }
        function q(a2, b2, c2) {
          if ("string" === typeof b2 && "" !== b2 || "number" === typeof b2) return b2 = Qg("" + b2, a2.mode, c2), b2.return = a2, b2;
          if ("object" === typeof b2 && null !== b2) {
            switch (b2.$$typeof) {
              case va:
                return c2 = Rg(b2.type, b2.key, b2.props, null, a2.mode, c2), c2.ref = Lg(a2, null, b2), c2.return = a2, c2;
              case wa:
                return b2 = Sg(b2, a2.mode, c2), b2.return = a2, b2;
              case Ha:
                var d2 = b2._init;
                return q(a2, d2(b2._payload), c2);
            }
            if (eb(b2) || Ka(b2)) return b2 = Tg(b2, a2.mode, c2, null), b2.return = a2, b2;
            Mg(a2, b2);
          }
          return null;
        }
        function r2(a2, b2, c2, d2) {
          var e2 = null !== b2 ? b2.key : null;
          if ("string" === typeof c2 && "" !== c2 || "number" === typeof c2) return null !== e2 ? null : h(a2, b2, "" + c2, d2);
          if ("object" === typeof c2 && null !== c2) {
            switch (c2.$$typeof) {
              case va:
                return c2.key === e2 ? k(a2, b2, c2, d2) : null;
              case wa:
                return c2.key === e2 ? l(a2, b2, c2, d2) : null;
              case Ha:
                return e2 = c2._init, r2(
                  a2,
                  b2,
                  e2(c2._payload),
                  d2
                );
            }
            if (eb(c2) || Ka(c2)) return null !== e2 ? null : m(a2, b2, c2, d2, null);
            Mg(a2, c2);
          }
          return null;
        }
        function y(a2, b2, c2, d2, e2) {
          if ("string" === typeof d2 && "" !== d2 || "number" === typeof d2) return a2 = a2.get(c2) || null, h(b2, a2, "" + d2, e2);
          if ("object" === typeof d2 && null !== d2) {
            switch (d2.$$typeof) {
              case va:
                return a2 = a2.get(null === d2.key ? c2 : d2.key) || null, k(b2, a2, d2, e2);
              case wa:
                return a2 = a2.get(null === d2.key ? c2 : d2.key) || null, l(b2, a2, d2, e2);
              case Ha:
                var f2 = d2._init;
                return y(a2, b2, c2, f2(d2._payload), e2);
            }
            if (eb(d2) || Ka(d2)) return a2 = a2.get(c2) || null, m(b2, a2, d2, e2, null);
            Mg(b2, d2);
          }
          return null;
        }
        function n(e2, g2, h2, k2) {
          for (var l2 = null, m2 = null, u = g2, w = g2 = 0, x = null; null !== u && w < h2.length; w++) {
            u.index > w ? (x = u, u = null) : x = u.sibling;
            var n2 = r2(e2, u, h2[w], k2);
            if (null === n2) {
              null === u && (u = x);
              break;
            }
            a && u && null === n2.alternate && b(e2, u);
            g2 = f(n2, g2, w);
            null === m2 ? l2 = n2 : m2.sibling = n2;
            m2 = n2;
            u = x;
          }
          if (w === h2.length) return c(e2, u), I && tg(e2, w), l2;
          if (null === u) {
            for (; w < h2.length; w++) u = q(e2, h2[w], k2), null !== u && (g2 = f(u, g2, w), null === m2 ? l2 = u : m2.sibling = u, m2 = u);
            I && tg(e2, w);
            return l2;
          }
          for (u = d(e2, u); w < h2.length; w++) x = y(u, e2, w, h2[w], k2), null !== x && (a && null !== x.alternate && u.delete(null === x.key ? w : x.key), g2 = f(x, g2, w), null === m2 ? l2 = x : m2.sibling = x, m2 = x);
          a && u.forEach(function(a2) {
            return b(e2, a2);
          });
          I && tg(e2, w);
          return l2;
        }
        function t(e2, g2, h2, k2) {
          var l2 = Ka(h2);
          if ("function" !== typeof l2) throw Error(p(150));
          h2 = l2.call(h2);
          if (null == h2) throw Error(p(151));
          for (var u = l2 = null, m2 = g2, w = g2 = 0, x = null, n2 = h2.next(); null !== m2 && !n2.done; w++, n2 = h2.next()) {
            m2.index > w ? (x = m2, m2 = null) : x = m2.sibling;
            var t2 = r2(e2, m2, n2.value, k2);
            if (null === t2) {
              null === m2 && (m2 = x);
              break;
            }
            a && m2 && null === t2.alternate && b(e2, m2);
            g2 = f(t2, g2, w);
            null === u ? l2 = t2 : u.sibling = t2;
            u = t2;
            m2 = x;
          }
          if (n2.done) return c(
            e2,
            m2
          ), I && tg(e2, w), l2;
          if (null === m2) {
            for (; !n2.done; w++, n2 = h2.next()) n2 = q(e2, n2.value, k2), null !== n2 && (g2 = f(n2, g2, w), null === u ? l2 = n2 : u.sibling = n2, u = n2);
            I && tg(e2, w);
            return l2;
          }
          for (m2 = d(e2, m2); !n2.done; w++, n2 = h2.next()) n2 = y(m2, e2, w, n2.value, k2), null !== n2 && (a && null !== n2.alternate && m2.delete(null === n2.key ? w : n2.key), g2 = f(n2, g2, w), null === u ? l2 = n2 : u.sibling = n2, u = n2);
          a && m2.forEach(function(a2) {
            return b(e2, a2);
          });
          I && tg(e2, w);
          return l2;
        }
        function J(a2, d2, f2, h2) {
          "object" === typeof f2 && null !== f2 && f2.type === ya && null === f2.key && (f2 = f2.props.children);
          if ("object" === typeof f2 && null !== f2) {
            switch (f2.$$typeof) {
              case va:
                a: {
                  for (var k2 = f2.key, l2 = d2; null !== l2; ) {
                    if (l2.key === k2) {
                      k2 = f2.type;
                      if (k2 === ya) {
                        if (7 === l2.tag) {
                          c(a2, l2.sibling);
                          d2 = e(l2, f2.props.children);
                          d2.return = a2;
                          a2 = d2;
                          break a;
                        }
                      } else if (l2.elementType === k2 || "object" === typeof k2 && null !== k2 && k2.$$typeof === Ha && Ng(k2) === l2.type) {
                        c(a2, l2.sibling);
                        d2 = e(l2, f2.props);
                        d2.ref = Lg(a2, l2, f2);
                        d2.return = a2;
                        a2 = d2;
                        break a;
                      }
                      c(a2, l2);
                      break;
                    } else b(a2, l2);
                    l2 = l2.sibling;
                  }
                  f2.type === ya ? (d2 = Tg(f2.props.children, a2.mode, h2, f2.key), d2.return = a2, a2 = d2) : (h2 = Rg(f2.type, f2.key, f2.props, null, a2.mode, h2), h2.ref = Lg(a2, d2, f2), h2.return = a2, a2 = h2);
                }
                return g(a2);
              case wa:
                a: {
                  for (l2 = f2.key; null !== d2; ) {
                    if (d2.key === l2) if (4 === d2.tag && d2.stateNode.containerInfo === f2.containerInfo && d2.stateNode.implementation === f2.implementation) {
                      c(a2, d2.sibling);
                      d2 = e(d2, f2.children || []);
                      d2.return = a2;
                      a2 = d2;
                      break a;
                    } else {
                      c(a2, d2);
                      break;
                    }
                    else b(a2, d2);
                    d2 = d2.sibling;
                  }
                  d2 = Sg(f2, a2.mode, h2);
                  d2.return = a2;
                  a2 = d2;
                }
                return g(a2);
              case Ha:
                return l2 = f2._init, J(a2, d2, l2(f2._payload), h2);
            }
            if (eb(f2)) return n(a2, d2, f2, h2);
            if (Ka(f2)) return t(a2, d2, f2, h2);
            Mg(a2, f2);
          }
          return "string" === typeof f2 && "" !== f2 || "number" === typeof f2 ? (f2 = "" + f2, null !== d2 && 6 === d2.tag ? (c(a2, d2.sibling), d2 = e(d2, f2), d2.return = a2, a2 = d2) : (c(a2, d2), d2 = Qg(f2, a2.mode, h2), d2.return = a2, a2 = d2), g(a2)) : c(a2, d2);
        }
        return J;
      }
      var Ug = Og(true);
      var Vg = Og(false);
      var Wg = Uf(null);
      var Xg = null;
      var Yg = null;
      var Zg = null;
      function $g() {
        Zg = Yg = Xg = null;
      }
      function ah(a) {
        var b = Wg.current;
        E(Wg);
        a._currentValue = b;
      }
      function bh(a, b, c) {
        for (; null !== a; ) {
          var d = a.alternate;
          (a.childLanes & b) !== b ? (a.childLanes |= b, null !== d && (d.childLanes |= b)) : null !== d && (d.childLanes & b) !== b && (d.childLanes |= b);
          if (a === c) break;
          a = a.return;
        }
      }
      function ch(a, b) {
        Xg = a;
        Zg = Yg = null;
        a = a.dependencies;
        null !== a && null !== a.firstContext && (0 !== (a.lanes & b) && (dh = true), a.firstContext = null);
      }
      function eh(a) {
        var b = a._currentValue;
        if (Zg !== a) if (a = { context: a, memoizedValue: b, next: null }, null === Yg) {
          if (null === Xg) throw Error(p(308));
          Yg = a;
          Xg.dependencies = { lanes: 0, firstContext: a };
        } else Yg = Yg.next = a;
        return b;
      }
      var fh = null;
      function gh(a) {
        null === fh ? fh = [a] : fh.push(a);
      }
      function hh(a, b, c, d) {
        var e = b.interleaved;
        null === e ? (c.next = c, gh(b)) : (c.next = e.next, e.next = c);
        b.interleaved = c;
        return ih(a, d);
      }
      function ih(a, b) {
        a.lanes |= b;
        var c = a.alternate;
        null !== c && (c.lanes |= b);
        c = a;
        for (a = a.return; null !== a; ) a.childLanes |= b, c = a.alternate, null !== c && (c.childLanes |= b), c = a, a = a.return;
        return 3 === c.tag ? c.stateNode : null;
      }
      var jh = false;
      function kh(a) {
        a.updateQueue = { baseState: a.memoizedState, firstBaseUpdate: null, lastBaseUpdate: null, shared: { pending: null, interleaved: null, lanes: 0 }, effects: null };
      }
      function lh(a, b) {
        a = a.updateQueue;
        b.updateQueue === a && (b.updateQueue = { baseState: a.baseState, firstBaseUpdate: a.firstBaseUpdate, lastBaseUpdate: a.lastBaseUpdate, shared: a.shared, effects: a.effects });
      }
      function mh(a, b) {
        return { eventTime: a, lane: b, tag: 0, payload: null, callback: null, next: null };
      }
      function nh(a, b, c) {
        var d = a.updateQueue;
        if (null === d) return null;
        d = d.shared;
        if (0 !== (K & 2)) {
          var e = d.pending;
          null === e ? b.next = b : (b.next = e.next, e.next = b);
          d.pending = b;
          return ih(a, c);
        }
        e = d.interleaved;
        null === e ? (b.next = b, gh(d)) : (b.next = e.next, e.next = b);
        d.interleaved = b;
        return ih(a, c);
      }
      function oh(a, b, c) {
        b = b.updateQueue;
        if (null !== b && (b = b.shared, 0 !== (c & 4194240))) {
          var d = b.lanes;
          d &= a.pendingLanes;
          c |= d;
          b.lanes = c;
          Cc(a, c);
        }
      }
      function ph(a, b) {
        var c = a.updateQueue, d = a.alternate;
        if (null !== d && (d = d.updateQueue, c === d)) {
          var e = null, f = null;
          c = c.firstBaseUpdate;
          if (null !== c) {
            do {
              var g = { eventTime: c.eventTime, lane: c.lane, tag: c.tag, payload: c.payload, callback: c.callback, next: null };
              null === f ? e = f = g : f = f.next = g;
              c = c.next;
            } while (null !== c);
            null === f ? e = f = b : f = f.next = b;
          } else e = f = b;
          c = { baseState: d.baseState, firstBaseUpdate: e, lastBaseUpdate: f, shared: d.shared, effects: d.effects };
          a.updateQueue = c;
          return;
        }
        a = c.lastBaseUpdate;
        null === a ? c.firstBaseUpdate = b : a.next = b;
        c.lastBaseUpdate = b;
      }
      function qh(a, b, c, d) {
        var e = a.updateQueue;
        jh = false;
        var f = e.firstBaseUpdate, g = e.lastBaseUpdate, h = e.shared.pending;
        if (null !== h) {
          e.shared.pending = null;
          var k = h, l = k.next;
          k.next = null;
          null === g ? f = l : g.next = l;
          g = k;
          var m = a.alternate;
          null !== m && (m = m.updateQueue, h = m.lastBaseUpdate, h !== g && (null === h ? m.firstBaseUpdate = l : h.next = l, m.lastBaseUpdate = k));
        }
        if (null !== f) {
          var q = e.baseState;
          g = 0;
          m = l = k = null;
          h = f;
          do {
            var r2 = h.lane, y = h.eventTime;
            if ((d & r2) === r2) {
              null !== m && (m = m.next = {
                eventTime: y,
                lane: 0,
                tag: h.tag,
                payload: h.payload,
                callback: h.callback,
                next: null
              });
              a: {
                var n = a, t = h;
                r2 = b;
                y = c;
                switch (t.tag) {
                  case 1:
                    n = t.payload;
                    if ("function" === typeof n) {
                      q = n.call(y, q, r2);
                      break a;
                    }
                    q = n;
                    break a;
                  case 3:
                    n.flags = n.flags & -65537 | 128;
                  case 0:
                    n = t.payload;
                    r2 = "function" === typeof n ? n.call(y, q, r2) : n;
                    if (null === r2 || void 0 === r2) break a;
                    q = A({}, q, r2);
                    break a;
                  case 2:
                    jh = true;
                }
              }
              null !== h.callback && 0 !== h.lane && (a.flags |= 64, r2 = e.effects, null === r2 ? e.effects = [h] : r2.push(h));
            } else y = { eventTime: y, lane: r2, tag: h.tag, payload: h.payload, callback: h.callback, next: null }, null === m ? (l = m = y, k = q) : m = m.next = y, g |= r2;
            h = h.next;
            if (null === h) if (h = e.shared.pending, null === h) break;
            else r2 = h, h = r2.next, r2.next = null, e.lastBaseUpdate = r2, e.shared.pending = null;
          } while (1);
          null === m && (k = q);
          e.baseState = k;
          e.firstBaseUpdate = l;
          e.lastBaseUpdate = m;
          b = e.shared.interleaved;
          if (null !== b) {
            e = b;
            do
              g |= e.lane, e = e.next;
            while (e !== b);
          } else null === f && (e.shared.lanes = 0);
          rh |= g;
          a.lanes = g;
          a.memoizedState = q;
        }
      }
      function sh(a, b, c) {
        a = b.effects;
        b.effects = null;
        if (null !== a) for (b = 0; b < a.length; b++) {
          var d = a[b], e = d.callback;
          if (null !== e) {
            d.callback = null;
            d = c;
            if ("function" !== typeof e) throw Error(p(191, e));
            e.call(d);
          }
        }
      }
      var th = {};
      var uh = Uf(th);
      var vh = Uf(th);
      var wh = Uf(th);
      function xh(a) {
        if (a === th) throw Error(p(174));
        return a;
      }
      function yh(a, b) {
        G(wh, b);
        G(vh, a);
        G(uh, th);
        a = b.nodeType;
        switch (a) {
          case 9:
          case 11:
            b = (b = b.documentElement) ? b.namespaceURI : lb(null, "");
            break;
          default:
            a = 8 === a ? b.parentNode : b, b = a.namespaceURI || null, a = a.tagName, b = lb(b, a);
        }
        E(uh);
        G(uh, b);
      }
      function zh() {
        E(uh);
        E(vh);
        E(wh);
      }
      function Ah(a) {
        xh(wh.current);
        var b = xh(uh.current);
        var c = lb(b, a.type);
        b !== c && (G(vh, a), G(uh, c));
      }
      function Bh(a) {
        vh.current === a && (E(uh), E(vh));
      }
      var L = Uf(0);
      function Ch(a) {
        for (var b = a; null !== b; ) {
          if (13 === b.tag) {
            var c = b.memoizedState;
            if (null !== c && (c = c.dehydrated, null === c || "$?" === c.data || "$!" === c.data)) return b;
          } else if (19 === b.tag && void 0 !== b.memoizedProps.revealOrder) {
            if (0 !== (b.flags & 128)) return b;
          } else if (null !== b.child) {
            b.child.return = b;
            b = b.child;
            continue;
          }
          if (b === a) break;
          for (; null === b.sibling; ) {
            if (null === b.return || b.return === a) return null;
            b = b.return;
          }
          b.sibling.return = b.return;
          b = b.sibling;
        }
        return null;
      }
      var Dh = [];
      function Eh() {
        for (var a = 0; a < Dh.length; a++) Dh[a]._workInProgressVersionPrimary = null;
        Dh.length = 0;
      }
      var Fh = ua.ReactCurrentDispatcher;
      var Gh = ua.ReactCurrentBatchConfig;
      var Hh = 0;
      var M = null;
      var N = null;
      var O = null;
      var Ih = false;
      var Jh = false;
      var Kh = 0;
      var Lh = 0;
      function P() {
        throw Error(p(321));
      }
      function Mh(a, b) {
        if (null === b) return false;
        for (var c = 0; c < b.length && c < a.length; c++) if (!He(a[c], b[c])) return false;
        return true;
      }
      function Nh(a, b, c, d, e, f) {
        Hh = f;
        M = b;
        b.memoizedState = null;
        b.updateQueue = null;
        b.lanes = 0;
        Fh.current = null === a || null === a.memoizedState ? Oh : Ph;
        a = c(d, e);
        if (Jh) {
          f = 0;
          do {
            Jh = false;
            Kh = 0;
            if (25 <= f) throw Error(p(301));
            f += 1;
            O = N = null;
            b.updateQueue = null;
            Fh.current = Qh;
            a = c(d, e);
          } while (Jh);
        }
        Fh.current = Rh;
        b = null !== N && null !== N.next;
        Hh = 0;
        O = N = M = null;
        Ih = false;
        if (b) throw Error(p(300));
        return a;
      }
      function Sh() {
        var a = 0 !== Kh;
        Kh = 0;
        return a;
      }
      function Th() {
        var a = { memoizedState: null, baseState: null, baseQueue: null, queue: null, next: null };
        null === O ? M.memoizedState = O = a : O = O.next = a;
        return O;
      }
      function Uh() {
        if (null === N) {
          var a = M.alternate;
          a = null !== a ? a.memoizedState : null;
        } else a = N.next;
        var b = null === O ? M.memoizedState : O.next;
        if (null !== b) O = b, N = a;
        else {
          if (null === a) throw Error(p(310));
          N = a;
          a = { memoizedState: N.memoizedState, baseState: N.baseState, baseQueue: N.baseQueue, queue: N.queue, next: null };
          null === O ? M.memoizedState = O = a : O = O.next = a;
        }
        return O;
      }
      function Vh(a, b) {
        return "function" === typeof b ? b(a) : b;
      }
      function Wh(a) {
        var b = Uh(), c = b.queue;
        if (null === c) throw Error(p(311));
        c.lastRenderedReducer = a;
        var d = N, e = d.baseQueue, f = c.pending;
        if (null !== f) {
          if (null !== e) {
            var g = e.next;
            e.next = f.next;
            f.next = g;
          }
          d.baseQueue = e = f;
          c.pending = null;
        }
        if (null !== e) {
          f = e.next;
          d = d.baseState;
          var h = g = null, k = null, l = f;
          do {
            var m = l.lane;
            if ((Hh & m) === m) null !== k && (k = k.next = { lane: 0, action: l.action, hasEagerState: l.hasEagerState, eagerState: l.eagerState, next: null }), d = l.hasEagerState ? l.eagerState : a(d, l.action);
            else {
              var q = {
                lane: m,
                action: l.action,
                hasEagerState: l.hasEagerState,
                eagerState: l.eagerState,
                next: null
              };
              null === k ? (h = k = q, g = d) : k = k.next = q;
              M.lanes |= m;
              rh |= m;
            }
            l = l.next;
          } while (null !== l && l !== f);
          null === k ? g = d : k.next = h;
          He(d, b.memoizedState) || (dh = true);
          b.memoizedState = d;
          b.baseState = g;
          b.baseQueue = k;
          c.lastRenderedState = d;
        }
        a = c.interleaved;
        if (null !== a) {
          e = a;
          do
            f = e.lane, M.lanes |= f, rh |= f, e = e.next;
          while (e !== a);
        } else null === e && (c.lanes = 0);
        return [b.memoizedState, c.dispatch];
      }
      function Xh(a) {
        var b = Uh(), c = b.queue;
        if (null === c) throw Error(p(311));
        c.lastRenderedReducer = a;
        var d = c.dispatch, e = c.pending, f = b.memoizedState;
        if (null !== e) {
          c.pending = null;
          var g = e = e.next;
          do
            f = a(f, g.action), g = g.next;
          while (g !== e);
          He(f, b.memoizedState) || (dh = true);
          b.memoizedState = f;
          null === b.baseQueue && (b.baseState = f);
          c.lastRenderedState = f;
        }
        return [f, d];
      }
      function Yh() {
      }
      function Zh(a, b) {
        var c = M, d = Uh(), e = b(), f = !He(d.memoizedState, e);
        f && (d.memoizedState = e, dh = true);
        d = d.queue;
        $h(ai.bind(null, c, d, a), [a]);
        if (d.getSnapshot !== b || f || null !== O && O.memoizedState.tag & 1) {
          c.flags |= 2048;
          bi(9, ci.bind(null, c, d, e, b), void 0, null);
          if (null === Q) throw Error(p(349));
          0 !== (Hh & 30) || di(c, b, e);
        }
        return e;
      }
      function di(a, b, c) {
        a.flags |= 16384;
        a = { getSnapshot: b, value: c };
        b = M.updateQueue;
        null === b ? (b = { lastEffect: null, stores: null }, M.updateQueue = b, b.stores = [a]) : (c = b.stores, null === c ? b.stores = [a] : c.push(a));
      }
      function ci(a, b, c, d) {
        b.value = c;
        b.getSnapshot = d;
        ei(b) && fi(a);
      }
      function ai(a, b, c) {
        return c(function() {
          ei(b) && fi(a);
        });
      }
      function ei(a) {
        var b = a.getSnapshot;
        a = a.value;
        try {
          var c = b();
          return !He(a, c);
        } catch (d) {
          return true;
        }
      }
      function fi(a) {
        var b = ih(a, 1);
        null !== b && gi(b, a, 1, -1);
      }
      function hi(a) {
        var b = Th();
        "function" === typeof a && (a = a());
        b.memoizedState = b.baseState = a;
        a = { pending: null, interleaved: null, lanes: 0, dispatch: null, lastRenderedReducer: Vh, lastRenderedState: a };
        b.queue = a;
        a = a.dispatch = ii.bind(null, M, a);
        return [b.memoizedState, a];
      }
      function bi(a, b, c, d) {
        a = { tag: a, create: b, destroy: c, deps: d, next: null };
        b = M.updateQueue;
        null === b ? (b = { lastEffect: null, stores: null }, M.updateQueue = b, b.lastEffect = a.next = a) : (c = b.lastEffect, null === c ? b.lastEffect = a.next = a : (d = c.next, c.next = a, a.next = d, b.lastEffect = a));
        return a;
      }
      function ji() {
        return Uh().memoizedState;
      }
      function ki(a, b, c, d) {
        var e = Th();
        M.flags |= a;
        e.memoizedState = bi(1 | b, c, void 0, void 0 === d ? null : d);
      }
      function li(a, b, c, d) {
        var e = Uh();
        d = void 0 === d ? null : d;
        var f = void 0;
        if (null !== N) {
          var g = N.memoizedState;
          f = g.destroy;
          if (null !== d && Mh(d, g.deps)) {
            e.memoizedState = bi(b, c, f, d);
            return;
          }
        }
        M.flags |= a;
        e.memoizedState = bi(1 | b, c, f, d);
      }
      function mi(a, b) {
        return ki(8390656, 8, a, b);
      }
      function $h(a, b) {
        return li(2048, 8, a, b);
      }
      function ni(a, b) {
        return li(4, 2, a, b);
      }
      function oi(a, b) {
        return li(4, 4, a, b);
      }
      function pi(a, b) {
        if ("function" === typeof b) return a = a(), b(a), function() {
          b(null);
        };
        if (null !== b && void 0 !== b) return a = a(), b.current = a, function() {
          b.current = null;
        };
      }
      function qi(a, b, c) {
        c = null !== c && void 0 !== c ? c.concat([a]) : null;
        return li(4, 4, pi.bind(null, b, a), c);
      }
      function ri() {
      }
      function si(a, b) {
        var c = Uh();
        b = void 0 === b ? null : b;
        var d = c.memoizedState;
        if (null !== d && null !== b && Mh(b, d[1])) return d[0];
        c.memoizedState = [a, b];
        return a;
      }
      function ti(a, b) {
        var c = Uh();
        b = void 0 === b ? null : b;
        var d = c.memoizedState;
        if (null !== d && null !== b && Mh(b, d[1])) return d[0];
        a = a();
        c.memoizedState = [a, b];
        return a;
      }
      function ui(a, b, c) {
        if (0 === (Hh & 21)) return a.baseState && (a.baseState = false, dh = true), a.memoizedState = c;
        He(c, b) || (c = yc(), M.lanes |= c, rh |= c, a.baseState = true);
        return b;
      }
      function vi(a, b) {
        var c = C;
        C = 0 !== c && 4 > c ? c : 4;
        a(true);
        var d = Gh.transition;
        Gh.transition = {};
        try {
          a(false), b();
        } finally {
          C = c, Gh.transition = d;
        }
      }
      function wi() {
        return Uh().memoizedState;
      }
      function xi(a, b, c) {
        var d = yi(a);
        c = { lane: d, action: c, hasEagerState: false, eagerState: null, next: null };
        if (zi(a)) Ai(b, c);
        else if (c = hh(a, b, c, d), null !== c) {
          var e = R();
          gi(c, a, d, e);
          Bi(c, b, d);
        }
      }
      function ii(a, b, c) {
        var d = yi(a), e = { lane: d, action: c, hasEagerState: false, eagerState: null, next: null };
        if (zi(a)) Ai(b, e);
        else {
          var f = a.alternate;
          if (0 === a.lanes && (null === f || 0 === f.lanes) && (f = b.lastRenderedReducer, null !== f)) try {
            var g = b.lastRenderedState, h = f(g, c);
            e.hasEagerState = true;
            e.eagerState = h;
            if (He(h, g)) {
              var k = b.interleaved;
              null === k ? (e.next = e, gh(b)) : (e.next = k.next, k.next = e);
              b.interleaved = e;
              return;
            }
          } catch (l) {
          } finally {
          }
          c = hh(a, b, e, d);
          null !== c && (e = R(), gi(c, a, d, e), Bi(c, b, d));
        }
      }
      function zi(a) {
        var b = a.alternate;
        return a === M || null !== b && b === M;
      }
      function Ai(a, b) {
        Jh = Ih = true;
        var c = a.pending;
        null === c ? b.next = b : (b.next = c.next, c.next = b);
        a.pending = b;
      }
      function Bi(a, b, c) {
        if (0 !== (c & 4194240)) {
          var d = b.lanes;
          d &= a.pendingLanes;
          c |= d;
          b.lanes = c;
          Cc(a, c);
        }
      }
      var Rh = { readContext: eh, useCallback: P, useContext: P, useEffect: P, useImperativeHandle: P, useInsertionEffect: P, useLayoutEffect: P, useMemo: P, useReducer: P, useRef: P, useState: P, useDebugValue: P, useDeferredValue: P, useTransition: P, useMutableSource: P, useSyncExternalStore: P, useId: P, unstable_isNewReconciler: false };
      var Oh = { readContext: eh, useCallback: function(a, b) {
        Th().memoizedState = [a, void 0 === b ? null : b];
        return a;
      }, useContext: eh, useEffect: mi, useImperativeHandle: function(a, b, c) {
        c = null !== c && void 0 !== c ? c.concat([a]) : null;
        return ki(
          4194308,
          4,
          pi.bind(null, b, a),
          c
        );
      }, useLayoutEffect: function(a, b) {
        return ki(4194308, 4, a, b);
      }, useInsertionEffect: function(a, b) {
        return ki(4, 2, a, b);
      }, useMemo: function(a, b) {
        var c = Th();
        b = void 0 === b ? null : b;
        a = a();
        c.memoizedState = [a, b];
        return a;
      }, useReducer: function(a, b, c) {
        var d = Th();
        b = void 0 !== c ? c(b) : b;
        d.memoizedState = d.baseState = b;
        a = { pending: null, interleaved: null, lanes: 0, dispatch: null, lastRenderedReducer: a, lastRenderedState: b };
        d.queue = a;
        a = a.dispatch = xi.bind(null, M, a);
        return [d.memoizedState, a];
      }, useRef: function(a) {
        var b = Th();
        a = { current: a };
        return b.memoizedState = a;
      }, useState: hi, useDebugValue: ri, useDeferredValue: function(a) {
        return Th().memoizedState = a;
      }, useTransition: function() {
        var a = hi(false), b = a[0];
        a = vi.bind(null, a[1]);
        Th().memoizedState = a;
        return [b, a];
      }, useMutableSource: function() {
      }, useSyncExternalStore: function(a, b, c) {
        var d = M, e = Th();
        if (I) {
          if (void 0 === c) throw Error(p(407));
          c = c();
        } else {
          c = b();
          if (null === Q) throw Error(p(349));
          0 !== (Hh & 30) || di(d, b, c);
        }
        e.memoizedState = c;
        var f = { value: c, getSnapshot: b };
        e.queue = f;
        mi(ai.bind(
          null,
          d,
          f,
          a
        ), [a]);
        d.flags |= 2048;
        bi(9, ci.bind(null, d, f, c, b), void 0, null);
        return c;
      }, useId: function() {
        var a = Th(), b = Q.identifierPrefix;
        if (I) {
          var c = sg;
          var d = rg;
          c = (d & ~(1 << 32 - oc(d) - 1)).toString(32) + c;
          b = ":" + b + "R" + c;
          c = Kh++;
          0 < c && (b += "H" + c.toString(32));
          b += ":";
        } else c = Lh++, b = ":" + b + "r" + c.toString(32) + ":";
        return a.memoizedState = b;
      }, unstable_isNewReconciler: false };
      var Ph = {
        readContext: eh,
        useCallback: si,
        useContext: eh,
        useEffect: $h,
        useImperativeHandle: qi,
        useInsertionEffect: ni,
        useLayoutEffect: oi,
        useMemo: ti,
        useReducer: Wh,
        useRef: ji,
        useState: function() {
          return Wh(Vh);
        },
        useDebugValue: ri,
        useDeferredValue: function(a) {
          var b = Uh();
          return ui(b, N.memoizedState, a);
        },
        useTransition: function() {
          var a = Wh(Vh)[0], b = Uh().memoizedState;
          return [a, b];
        },
        useMutableSource: Yh,
        useSyncExternalStore: Zh,
        useId: wi,
        unstable_isNewReconciler: false
      };
      var Qh = { readContext: eh, useCallback: si, useContext: eh, useEffect: $h, useImperativeHandle: qi, useInsertionEffect: ni, useLayoutEffect: oi, useMemo: ti, useReducer: Xh, useRef: ji, useState: function() {
        return Xh(Vh);
      }, useDebugValue: ri, useDeferredValue: function(a) {
        var b = Uh();
        return null === N ? b.memoizedState = a : ui(b, N.memoizedState, a);
      }, useTransition: function() {
        var a = Xh(Vh)[0], b = Uh().memoizedState;
        return [a, b];
      }, useMutableSource: Yh, useSyncExternalStore: Zh, useId: wi, unstable_isNewReconciler: false };
      function Ci(a, b) {
        if (a && a.defaultProps) {
          b = A({}, b);
          a = a.defaultProps;
          for (var c in a) void 0 === b[c] && (b[c] = a[c]);
          return b;
        }
        return b;
      }
      function Di(a, b, c, d) {
        b = a.memoizedState;
        c = c(d, b);
        c = null === c || void 0 === c ? b : A({}, b, c);
        a.memoizedState = c;
        0 === a.lanes && (a.updateQueue.baseState = c);
      }
      var Ei = { isMounted: function(a) {
        return (a = a._reactInternals) ? Vb(a) === a : false;
      }, enqueueSetState: function(a, b, c) {
        a = a._reactInternals;
        var d = R(), e = yi(a), f = mh(d, e);
        f.payload = b;
        void 0 !== c && null !== c && (f.callback = c);
        b = nh(a, f, e);
        null !== b && (gi(b, a, e, d), oh(b, a, e));
      }, enqueueReplaceState: function(a, b, c) {
        a = a._reactInternals;
        var d = R(), e = yi(a), f = mh(d, e);
        f.tag = 1;
        f.payload = b;
        void 0 !== c && null !== c && (f.callback = c);
        b = nh(a, f, e);
        null !== b && (gi(b, a, e, d), oh(b, a, e));
      }, enqueueForceUpdate: function(a, b) {
        a = a._reactInternals;
        var c = R(), d = yi(a), e = mh(c, d);
        e.tag = 2;
        void 0 !== b && null !== b && (e.callback = b);
        b = nh(a, e, d);
        null !== b && (gi(b, a, d, c), oh(b, a, d));
      } };
      function Fi(a, b, c, d, e, f, g) {
        a = a.stateNode;
        return "function" === typeof a.shouldComponentUpdate ? a.shouldComponentUpdate(d, f, g) : b.prototype && b.prototype.isPureReactComponent ? !Ie(c, d) || !Ie(e, f) : true;
      }
      function Gi(a, b, c) {
        var d = false, e = Vf;
        var f = b.contextType;
        "object" === typeof f && null !== f ? f = eh(f) : (e = Zf(b) ? Xf : H.current, d = b.contextTypes, f = (d = null !== d && void 0 !== d) ? Yf(a, e) : Vf);
        b = new b(c, f);
        a.memoizedState = null !== b.state && void 0 !== b.state ? b.state : null;
        b.updater = Ei;
        a.stateNode = b;
        b._reactInternals = a;
        d && (a = a.stateNode, a.__reactInternalMemoizedUnmaskedChildContext = e, a.__reactInternalMemoizedMaskedChildContext = f);
        return b;
      }
      function Hi(a, b, c, d) {
        a = b.state;
        "function" === typeof b.componentWillReceiveProps && b.componentWillReceiveProps(c, d);
        "function" === typeof b.UNSAFE_componentWillReceiveProps && b.UNSAFE_componentWillReceiveProps(c, d);
        b.state !== a && Ei.enqueueReplaceState(b, b.state, null);
      }
      function Ii(a, b, c, d) {
        var e = a.stateNode;
        e.props = c;
        e.state = a.memoizedState;
        e.refs = {};
        kh(a);
        var f = b.contextType;
        "object" === typeof f && null !== f ? e.context = eh(f) : (f = Zf(b) ? Xf : H.current, e.context = Yf(a, f));
        e.state = a.memoizedState;
        f = b.getDerivedStateFromProps;
        "function" === typeof f && (Di(a, b, f, c), e.state = a.memoizedState);
        "function" === typeof b.getDerivedStateFromProps || "function" === typeof e.getSnapshotBeforeUpdate || "function" !== typeof e.UNSAFE_componentWillMount && "function" !== typeof e.componentWillMount || (b = e.state, "function" === typeof e.componentWillMount && e.componentWillMount(), "function" === typeof e.UNSAFE_componentWillMount && e.UNSAFE_componentWillMount(), b !== e.state && Ei.enqueueReplaceState(e, e.state, null), qh(a, c, e, d), e.state = a.memoizedState);
        "function" === typeof e.componentDidMount && (a.flags |= 4194308);
      }
      function Ji(a, b) {
        try {
          var c = "", d = b;
          do
            c += Pa(d), d = d.return;
          while (d);
          var e = c;
        } catch (f) {
          e = "\nError generating stack: " + f.message + "\n" + f.stack;
        }
        return { value: a, source: b, stack: e, digest: null };
      }
      function Ki(a, b, c) {
        return { value: a, source: null, stack: null != c ? c : null, digest: null != b ? b : null };
      }
      function Li(a, b) {
        try {
          console.error(b.value);
        } catch (c) {
          setTimeout(function() {
            throw c;
          });
        }
      }
      var Mi = "function" === typeof WeakMap ? WeakMap : Map;
      function Ni(a, b, c) {
        c = mh(-1, c);
        c.tag = 3;
        c.payload = { element: null };
        var d = b.value;
        c.callback = function() {
          Oi || (Oi = true, Pi = d);
          Li(a, b);
        };
        return c;
      }
      function Qi(a, b, c) {
        c = mh(-1, c);
        c.tag = 3;
        var d = a.type.getDerivedStateFromError;
        if ("function" === typeof d) {
          var e = b.value;
          c.payload = function() {
            return d(e);
          };
          c.callback = function() {
            Li(a, b);
          };
        }
        var f = a.stateNode;
        null !== f && "function" === typeof f.componentDidCatch && (c.callback = function() {
          Li(a, b);
          "function" !== typeof d && (null === Ri ? Ri = /* @__PURE__ */ new Set([this]) : Ri.add(this));
          var c2 = b.stack;
          this.componentDidCatch(b.value, { componentStack: null !== c2 ? c2 : "" });
        });
        return c;
      }
      function Si(a, b, c) {
        var d = a.pingCache;
        if (null === d) {
          d = a.pingCache = new Mi();
          var e = /* @__PURE__ */ new Set();
          d.set(b, e);
        } else e = d.get(b), void 0 === e && (e = /* @__PURE__ */ new Set(), d.set(b, e));
        e.has(c) || (e.add(c), a = Ti.bind(null, a, b, c), b.then(a, a));
      }
      function Ui(a) {
        do {
          var b;
          if (b = 13 === a.tag) b = a.memoizedState, b = null !== b ? null !== b.dehydrated ? true : false : true;
          if (b) return a;
          a = a.return;
        } while (null !== a);
        return null;
      }
      function Vi(a, b, c, d, e) {
        if (0 === (a.mode & 1)) return a === b ? a.flags |= 65536 : (a.flags |= 128, c.flags |= 131072, c.flags &= -52805, 1 === c.tag && (null === c.alternate ? c.tag = 17 : (b = mh(-1, 1), b.tag = 2, nh(c, b, 1))), c.lanes |= 1), a;
        a.flags |= 65536;
        a.lanes = e;
        return a;
      }
      var Wi = ua.ReactCurrentOwner;
      var dh = false;
      function Xi(a, b, c, d) {
        b.child = null === a ? Vg(b, null, c, d) : Ug(b, a.child, c, d);
      }
      function Yi(a, b, c, d, e) {
        c = c.render;
        var f = b.ref;
        ch(b, e);
        d = Nh(a, b, c, d, f, e);
        c = Sh();
        if (null !== a && !dh) return b.updateQueue = a.updateQueue, b.flags &= -2053, a.lanes &= ~e, Zi(a, b, e);
        I && c && vg(b);
        b.flags |= 1;
        Xi(a, b, d, e);
        return b.child;
      }
      function $i(a, b, c, d, e) {
        if (null === a) {
          var f = c.type;
          if ("function" === typeof f && !aj(f) && void 0 === f.defaultProps && null === c.compare && void 0 === c.defaultProps) return b.tag = 15, b.type = f, bj(a, b, f, d, e);
          a = Rg(c.type, null, d, b, b.mode, e);
          a.ref = b.ref;
          a.return = b;
          return b.child = a;
        }
        f = a.child;
        if (0 === (a.lanes & e)) {
          var g = f.memoizedProps;
          c = c.compare;
          c = null !== c ? c : Ie;
          if (c(g, d) && a.ref === b.ref) return Zi(a, b, e);
        }
        b.flags |= 1;
        a = Pg(f, d);
        a.ref = b.ref;
        a.return = b;
        return b.child = a;
      }
      function bj(a, b, c, d, e) {
        if (null !== a) {
          var f = a.memoizedProps;
          if (Ie(f, d) && a.ref === b.ref) if (dh = false, b.pendingProps = d = f, 0 !== (a.lanes & e)) 0 !== (a.flags & 131072) && (dh = true);
          else return b.lanes = a.lanes, Zi(a, b, e);
        }
        return cj(a, b, c, d, e);
      }
      function dj(a, b, c) {
        var d = b.pendingProps, e = d.children, f = null !== a ? a.memoizedState : null;
        if ("hidden" === d.mode) if (0 === (b.mode & 1)) b.memoizedState = { baseLanes: 0, cachePool: null, transitions: null }, G(ej, fj), fj |= c;
        else {
          if (0 === (c & 1073741824)) return a = null !== f ? f.baseLanes | c : c, b.lanes = b.childLanes = 1073741824, b.memoizedState = { baseLanes: a, cachePool: null, transitions: null }, b.updateQueue = null, G(ej, fj), fj |= a, null;
          b.memoizedState = { baseLanes: 0, cachePool: null, transitions: null };
          d = null !== f ? f.baseLanes : c;
          G(ej, fj);
          fj |= d;
        }
        else null !== f ? (d = f.baseLanes | c, b.memoizedState = null) : d = c, G(ej, fj), fj |= d;
        Xi(a, b, e, c);
        return b.child;
      }
      function gj(a, b) {
        var c = b.ref;
        if (null === a && null !== c || null !== a && a.ref !== c) b.flags |= 512, b.flags |= 2097152;
      }
      function cj(a, b, c, d, e) {
        var f = Zf(c) ? Xf : H.current;
        f = Yf(b, f);
        ch(b, e);
        c = Nh(a, b, c, d, f, e);
        d = Sh();
        if (null !== a && !dh) return b.updateQueue = a.updateQueue, b.flags &= -2053, a.lanes &= ~e, Zi(a, b, e);
        I && d && vg(b);
        b.flags |= 1;
        Xi(a, b, c, e);
        return b.child;
      }
      function hj(a, b, c, d, e) {
        if (Zf(c)) {
          var f = true;
          cg(b);
        } else f = false;
        ch(b, e);
        if (null === b.stateNode) ij(a, b), Gi(b, c, d), Ii(b, c, d, e), d = true;
        else if (null === a) {
          var g = b.stateNode, h = b.memoizedProps;
          g.props = h;
          var k = g.context, l = c.contextType;
          "object" === typeof l && null !== l ? l = eh(l) : (l = Zf(c) ? Xf : H.current, l = Yf(b, l));
          var m = c.getDerivedStateFromProps, q = "function" === typeof m || "function" === typeof g.getSnapshotBeforeUpdate;
          q || "function" !== typeof g.UNSAFE_componentWillReceiveProps && "function" !== typeof g.componentWillReceiveProps || (h !== d || k !== l) && Hi(b, g, d, l);
          jh = false;
          var r2 = b.memoizedState;
          g.state = r2;
          qh(b, d, g, e);
          k = b.memoizedState;
          h !== d || r2 !== k || Wf.current || jh ? ("function" === typeof m && (Di(b, c, m, d), k = b.memoizedState), (h = jh || Fi(b, c, h, d, r2, k, l)) ? (q || "function" !== typeof g.UNSAFE_componentWillMount && "function" !== typeof g.componentWillMount || ("function" === typeof g.componentWillMount && g.componentWillMount(), "function" === typeof g.UNSAFE_componentWillMount && g.UNSAFE_componentWillMount()), "function" === typeof g.componentDidMount && (b.flags |= 4194308)) : ("function" === typeof g.componentDidMount && (b.flags |= 4194308), b.memoizedProps = d, b.memoizedState = k), g.props = d, g.state = k, g.context = l, d = h) : ("function" === typeof g.componentDidMount && (b.flags |= 4194308), d = false);
        } else {
          g = b.stateNode;
          lh(a, b);
          h = b.memoizedProps;
          l = b.type === b.elementType ? h : Ci(b.type, h);
          g.props = l;
          q = b.pendingProps;
          r2 = g.context;
          k = c.contextType;
          "object" === typeof k && null !== k ? k = eh(k) : (k = Zf(c) ? Xf : H.current, k = Yf(b, k));
          var y = c.getDerivedStateFromProps;
          (m = "function" === typeof y || "function" === typeof g.getSnapshotBeforeUpdate) || "function" !== typeof g.UNSAFE_componentWillReceiveProps && "function" !== typeof g.componentWillReceiveProps || (h !== q || r2 !== k) && Hi(b, g, d, k);
          jh = false;
          r2 = b.memoizedState;
          g.state = r2;
          qh(b, d, g, e);
          var n = b.memoizedState;
          h !== q || r2 !== n || Wf.current || jh ? ("function" === typeof y && (Di(b, c, y, d), n = b.memoizedState), (l = jh || Fi(b, c, l, d, r2, n, k) || false) ? (m || "function" !== typeof g.UNSAFE_componentWillUpdate && "function" !== typeof g.componentWillUpdate || ("function" === typeof g.componentWillUpdate && g.componentWillUpdate(d, n, k), "function" === typeof g.UNSAFE_componentWillUpdate && g.UNSAFE_componentWillUpdate(d, n, k)), "function" === typeof g.componentDidUpdate && (b.flags |= 4), "function" === typeof g.getSnapshotBeforeUpdate && (b.flags |= 1024)) : ("function" !== typeof g.componentDidUpdate || h === a.memoizedProps && r2 === a.memoizedState || (b.flags |= 4), "function" !== typeof g.getSnapshotBeforeUpdate || h === a.memoizedProps && r2 === a.memoizedState || (b.flags |= 1024), b.memoizedProps = d, b.memoizedState = n), g.props = d, g.state = n, g.context = k, d = l) : ("function" !== typeof g.componentDidUpdate || h === a.memoizedProps && r2 === a.memoizedState || (b.flags |= 4), "function" !== typeof g.getSnapshotBeforeUpdate || h === a.memoizedProps && r2 === a.memoizedState || (b.flags |= 1024), d = false);
        }
        return jj(a, b, c, d, f, e);
      }
      function jj(a, b, c, d, e, f) {
        gj(a, b);
        var g = 0 !== (b.flags & 128);
        if (!d && !g) return e && dg(b, c, false), Zi(a, b, f);
        d = b.stateNode;
        Wi.current = b;
        var h = g && "function" !== typeof c.getDerivedStateFromError ? null : d.render();
        b.flags |= 1;
        null !== a && g ? (b.child = Ug(b, a.child, null, f), b.child = Ug(b, null, h, f)) : Xi(a, b, h, f);
        b.memoizedState = d.state;
        e && dg(b, c, true);
        return b.child;
      }
      function kj(a) {
        var b = a.stateNode;
        b.pendingContext ? ag(a, b.pendingContext, b.pendingContext !== b.context) : b.context && ag(a, b.context, false);
        yh(a, b.containerInfo);
      }
      function lj(a, b, c, d, e) {
        Ig();
        Jg(e);
        b.flags |= 256;
        Xi(a, b, c, d);
        return b.child;
      }
      var mj = { dehydrated: null, treeContext: null, retryLane: 0 };
      function nj(a) {
        return { baseLanes: a, cachePool: null, transitions: null };
      }
      function oj(a, b, c) {
        var d = b.pendingProps, e = L.current, f = false, g = 0 !== (b.flags & 128), h;
        (h = g) || (h = null !== a && null === a.memoizedState ? false : 0 !== (e & 2));
        if (h) f = true, b.flags &= -129;
        else if (null === a || null !== a.memoizedState) e |= 1;
        G(L, e & 1);
        if (null === a) {
          Eg(b);
          a = b.memoizedState;
          if (null !== a && (a = a.dehydrated, null !== a)) return 0 === (b.mode & 1) ? b.lanes = 1 : "$!" === a.data ? b.lanes = 8 : b.lanes = 1073741824, null;
          g = d.children;
          a = d.fallback;
          return f ? (d = b.mode, f = b.child, g = { mode: "hidden", children: g }, 0 === (d & 1) && null !== f ? (f.childLanes = 0, f.pendingProps = g) : f = pj(g, d, 0, null), a = Tg(a, d, c, null), f.return = b, a.return = b, f.sibling = a, b.child = f, b.child.memoizedState = nj(c), b.memoizedState = mj, a) : qj(b, g);
        }
        e = a.memoizedState;
        if (null !== e && (h = e.dehydrated, null !== h)) return rj(a, b, g, d, h, e, c);
        if (f) {
          f = d.fallback;
          g = b.mode;
          e = a.child;
          h = e.sibling;
          var k = { mode: "hidden", children: d.children };
          0 === (g & 1) && b.child !== e ? (d = b.child, d.childLanes = 0, d.pendingProps = k, b.deletions = null) : (d = Pg(e, k), d.subtreeFlags = e.subtreeFlags & 14680064);
          null !== h ? f = Pg(h, f) : (f = Tg(f, g, c, null), f.flags |= 2);
          f.return = b;
          d.return = b;
          d.sibling = f;
          b.child = d;
          d = f;
          f = b.child;
          g = a.child.memoizedState;
          g = null === g ? nj(c) : { baseLanes: g.baseLanes | c, cachePool: null, transitions: g.transitions };
          f.memoizedState = g;
          f.childLanes = a.childLanes & ~c;
          b.memoizedState = mj;
          return d;
        }
        f = a.child;
        a = f.sibling;
        d = Pg(f, { mode: "visible", children: d.children });
        0 === (b.mode & 1) && (d.lanes = c);
        d.return = b;
        d.sibling = null;
        null !== a && (c = b.deletions, null === c ? (b.deletions = [a], b.flags |= 16) : c.push(a));
        b.child = d;
        b.memoizedState = null;
        return d;
      }
      function qj(a, b) {
        b = pj({ mode: "visible", children: b }, a.mode, 0, null);
        b.return = a;
        return a.child = b;
      }
      function sj(a, b, c, d) {
        null !== d && Jg(d);
        Ug(b, a.child, null, c);
        a = qj(b, b.pendingProps.children);
        a.flags |= 2;
        b.memoizedState = null;
        return a;
      }
      function rj(a, b, c, d, e, f, g) {
        if (c) {
          if (b.flags & 256) return b.flags &= -257, d = Ki(Error(p(422))), sj(a, b, g, d);
          if (null !== b.memoizedState) return b.child = a.child, b.flags |= 128, null;
          f = d.fallback;
          e = b.mode;
          d = pj({ mode: "visible", children: d.children }, e, 0, null);
          f = Tg(f, e, g, null);
          f.flags |= 2;
          d.return = b;
          f.return = b;
          d.sibling = f;
          b.child = d;
          0 !== (b.mode & 1) && Ug(b, a.child, null, g);
          b.child.memoizedState = nj(g);
          b.memoizedState = mj;
          return f;
        }
        if (0 === (b.mode & 1)) return sj(a, b, g, null);
        if ("$!" === e.data) {
          d = e.nextSibling && e.nextSibling.dataset;
          if (d) var h = d.dgst;
          d = h;
          f = Error(p(419));
          d = Ki(f, d, void 0);
          return sj(a, b, g, d);
        }
        h = 0 !== (g & a.childLanes);
        if (dh || h) {
          d = Q;
          if (null !== d) {
            switch (g & -g) {
              case 4:
                e = 2;
                break;
              case 16:
                e = 8;
                break;
              case 64:
              case 128:
              case 256:
              case 512:
              case 1024:
              case 2048:
              case 4096:
              case 8192:
              case 16384:
              case 32768:
              case 65536:
              case 131072:
              case 262144:
              case 524288:
              case 1048576:
              case 2097152:
              case 4194304:
              case 8388608:
              case 16777216:
              case 33554432:
              case 67108864:
                e = 32;
                break;
              case 536870912:
                e = 268435456;
                break;
              default:
                e = 0;
            }
            e = 0 !== (e & (d.suspendedLanes | g)) ? 0 : e;
            0 !== e && e !== f.retryLane && (f.retryLane = e, ih(a, e), gi(d, a, e, -1));
          }
          tj();
          d = Ki(Error(p(421)));
          return sj(a, b, g, d);
        }
        if ("$?" === e.data) return b.flags |= 128, b.child = a.child, b = uj.bind(null, a), e._reactRetry = b, null;
        a = f.treeContext;
        yg = Lf(e.nextSibling);
        xg = b;
        I = true;
        zg = null;
        null !== a && (og[pg++] = rg, og[pg++] = sg, og[pg++] = qg, rg = a.id, sg = a.overflow, qg = b);
        b = qj(b, d.children);
        b.flags |= 4096;
        return b;
      }
      function vj(a, b, c) {
        a.lanes |= b;
        var d = a.alternate;
        null !== d && (d.lanes |= b);
        bh(a.return, b, c);
      }
      function wj(a, b, c, d, e) {
        var f = a.memoizedState;
        null === f ? a.memoizedState = { isBackwards: b, rendering: null, renderingStartTime: 0, last: d, tail: c, tailMode: e } : (f.isBackwards = b, f.rendering = null, f.renderingStartTime = 0, f.last = d, f.tail = c, f.tailMode = e);
      }
      function xj(a, b, c) {
        var d = b.pendingProps, e = d.revealOrder, f = d.tail;
        Xi(a, b, d.children, c);
        d = L.current;
        if (0 !== (d & 2)) d = d & 1 | 2, b.flags |= 128;
        else {
          if (null !== a && 0 !== (a.flags & 128)) a: for (a = b.child; null !== a; ) {
            if (13 === a.tag) null !== a.memoizedState && vj(a, c, b);
            else if (19 === a.tag) vj(a, c, b);
            else if (null !== a.child) {
              a.child.return = a;
              a = a.child;
              continue;
            }
            if (a === b) break a;
            for (; null === a.sibling; ) {
              if (null === a.return || a.return === b) break a;
              a = a.return;
            }
            a.sibling.return = a.return;
            a = a.sibling;
          }
          d &= 1;
        }
        G(L, d);
        if (0 === (b.mode & 1)) b.memoizedState = null;
        else switch (e) {
          case "forwards":
            c = b.child;
            for (e = null; null !== c; ) a = c.alternate, null !== a && null === Ch(a) && (e = c), c = c.sibling;
            c = e;
            null === c ? (e = b.child, b.child = null) : (e = c.sibling, c.sibling = null);
            wj(b, false, e, c, f);
            break;
          case "backwards":
            c = null;
            e = b.child;
            for (b.child = null; null !== e; ) {
              a = e.alternate;
              if (null !== a && null === Ch(a)) {
                b.child = e;
                break;
              }
              a = e.sibling;
              e.sibling = c;
              c = e;
              e = a;
            }
            wj(b, true, c, null, f);
            break;
          case "together":
            wj(b, false, null, null, void 0);
            break;
          default:
            b.memoizedState = null;
        }
        return b.child;
      }
      function ij(a, b) {
        0 === (b.mode & 1) && null !== a && (a.alternate = null, b.alternate = null, b.flags |= 2);
      }
      function Zi(a, b, c) {
        null !== a && (b.dependencies = a.dependencies);
        rh |= b.lanes;
        if (0 === (c & b.childLanes)) return null;
        if (null !== a && b.child !== a.child) throw Error(p(153));
        if (null !== b.child) {
          a = b.child;
          c = Pg(a, a.pendingProps);
          b.child = c;
          for (c.return = b; null !== a.sibling; ) a = a.sibling, c = c.sibling = Pg(a, a.pendingProps), c.return = b;
          c.sibling = null;
        }
        return b.child;
      }
      function yj(a, b, c) {
        switch (b.tag) {
          case 3:
            kj(b);
            Ig();
            break;
          case 5:
            Ah(b);
            break;
          case 1:
            Zf(b.type) && cg(b);
            break;
          case 4:
            yh(b, b.stateNode.containerInfo);
            break;
          case 10:
            var d = b.type._context, e = b.memoizedProps.value;
            G(Wg, d._currentValue);
            d._currentValue = e;
            break;
          case 13:
            d = b.memoizedState;
            if (null !== d) {
              if (null !== d.dehydrated) return G(L, L.current & 1), b.flags |= 128, null;
              if (0 !== (c & b.child.childLanes)) return oj(a, b, c);
              G(L, L.current & 1);
              a = Zi(a, b, c);
              return null !== a ? a.sibling : null;
            }
            G(L, L.current & 1);
            break;
          case 19:
            d = 0 !== (c & b.childLanes);
            if (0 !== (a.flags & 128)) {
              if (d) return xj(a, b, c);
              b.flags |= 128;
            }
            e = b.memoizedState;
            null !== e && (e.rendering = null, e.tail = null, e.lastEffect = null);
            G(L, L.current);
            if (d) break;
            else return null;
          case 22:
          case 23:
            return b.lanes = 0, dj(a, b, c);
        }
        return Zi(a, b, c);
      }
      var zj;
      var Aj;
      var Bj;
      var Cj;
      zj = function(a, b) {
        for (var c = b.child; null !== c; ) {
          if (5 === c.tag || 6 === c.tag) a.appendChild(c.stateNode);
          else if (4 !== c.tag && null !== c.child) {
            c.child.return = c;
            c = c.child;
            continue;
          }
          if (c === b) break;
          for (; null === c.sibling; ) {
            if (null === c.return || c.return === b) return;
            c = c.return;
          }
          c.sibling.return = c.return;
          c = c.sibling;
        }
      };
      Aj = function() {
      };
      Bj = function(a, b, c, d) {
        var e = a.memoizedProps;
        if (e !== d) {
          a = b.stateNode;
          xh(uh.current);
          var f = null;
          switch (c) {
            case "input":
              e = Ya(a, e);
              d = Ya(a, d);
              f = [];
              break;
            case "select":
              e = A({}, e, { value: void 0 });
              d = A({}, d, { value: void 0 });
              f = [];
              break;
            case "textarea":
              e = gb(a, e);
              d = gb(a, d);
              f = [];
              break;
            default:
              "function" !== typeof e.onClick && "function" === typeof d.onClick && (a.onclick = Bf);
          }
          ub(c, d);
          var g;
          c = null;
          for (l in e) if (!d.hasOwnProperty(l) && e.hasOwnProperty(l) && null != e[l]) if ("style" === l) {
            var h = e[l];
            for (g in h) h.hasOwnProperty(g) && (c || (c = {}), c[g] = "");
          } else "dangerouslySetInnerHTML" !== l && "children" !== l && "suppressContentEditableWarning" !== l && "suppressHydrationWarning" !== l && "autoFocus" !== l && (ea.hasOwnProperty(l) ? f || (f = []) : (f = f || []).push(l, null));
          for (l in d) {
            var k = d[l];
            h = null != e ? e[l] : void 0;
            if (d.hasOwnProperty(l) && k !== h && (null != k || null != h)) if ("style" === l) if (h) {
              for (g in h) !h.hasOwnProperty(g) || k && k.hasOwnProperty(g) || (c || (c = {}), c[g] = "");
              for (g in k) k.hasOwnProperty(g) && h[g] !== k[g] && (c || (c = {}), c[g] = k[g]);
            } else c || (f || (f = []), f.push(
              l,
              c
            )), c = k;
            else "dangerouslySetInnerHTML" === l ? (k = k ? k.__html : void 0, h = h ? h.__html : void 0, null != k && h !== k && (f = f || []).push(l, k)) : "children" === l ? "string" !== typeof k && "number" !== typeof k || (f = f || []).push(l, "" + k) : "suppressContentEditableWarning" !== l && "suppressHydrationWarning" !== l && (ea.hasOwnProperty(l) ? (null != k && "onScroll" === l && D("scroll", a), f || h === k || (f = [])) : (f = f || []).push(l, k));
          }
          c && (f = f || []).push("style", c);
          var l = f;
          if (b.updateQueue = l) b.flags |= 4;
        }
      };
      Cj = function(a, b, c, d) {
        c !== d && (b.flags |= 4);
      };
      function Dj(a, b) {
        if (!I) switch (a.tailMode) {
          case "hidden":
            b = a.tail;
            for (var c = null; null !== b; ) null !== b.alternate && (c = b), b = b.sibling;
            null === c ? a.tail = null : c.sibling = null;
            break;
          case "collapsed":
            c = a.tail;
            for (var d = null; null !== c; ) null !== c.alternate && (d = c), c = c.sibling;
            null === d ? b || null === a.tail ? a.tail = null : a.tail.sibling = null : d.sibling = null;
        }
      }
      function S(a) {
        var b = null !== a.alternate && a.alternate.child === a.child, c = 0, d = 0;
        if (b) for (var e = a.child; null !== e; ) c |= e.lanes | e.childLanes, d |= e.subtreeFlags & 14680064, d |= e.flags & 14680064, e.return = a, e = e.sibling;
        else for (e = a.child; null !== e; ) c |= e.lanes | e.childLanes, d |= e.subtreeFlags, d |= e.flags, e.return = a, e = e.sibling;
        a.subtreeFlags |= d;
        a.childLanes = c;
        return b;
      }
      function Ej(a, b, c) {
        var d = b.pendingProps;
        wg(b);
        switch (b.tag) {
          case 2:
          case 16:
          case 15:
          case 0:
          case 11:
          case 7:
          case 8:
          case 12:
          case 9:
          case 14:
            return S(b), null;
          case 1:
            return Zf(b.type) && $f(), S(b), null;
          case 3:
            d = b.stateNode;
            zh();
            E(Wf);
            E(H);
            Eh();
            d.pendingContext && (d.context = d.pendingContext, d.pendingContext = null);
            if (null === a || null === a.child) Gg(b) ? b.flags |= 4 : null === a || a.memoizedState.isDehydrated && 0 === (b.flags & 256) || (b.flags |= 1024, null !== zg && (Fj(zg), zg = null));
            Aj(a, b);
            S(b);
            return null;
          case 5:
            Bh(b);
            var e = xh(wh.current);
            c = b.type;
            if (null !== a && null != b.stateNode) Bj(a, b, c, d, e), a.ref !== b.ref && (b.flags |= 512, b.flags |= 2097152);
            else {
              if (!d) {
                if (null === b.stateNode) throw Error(p(166));
                S(b);
                return null;
              }
              a = xh(uh.current);
              if (Gg(b)) {
                d = b.stateNode;
                c = b.type;
                var f = b.memoizedProps;
                d[Of] = b;
                d[Pf] = f;
                a = 0 !== (b.mode & 1);
                switch (c) {
                  case "dialog":
                    D("cancel", d);
                    D("close", d);
                    break;
                  case "iframe":
                  case "object":
                  case "embed":
                    D("load", d);
                    break;
                  case "video":
                  case "audio":
                    for (e = 0; e < lf.length; e++) D(lf[e], d);
                    break;
                  case "source":
                    D("error", d);
                    break;
                  case "img":
                  case "image":
                  case "link":
                    D(
                      "error",
                      d
                    );
                    D("load", d);
                    break;
                  case "details":
                    D("toggle", d);
                    break;
                  case "input":
                    Za(d, f);
                    D("invalid", d);
                    break;
                  case "select":
                    d._wrapperState = { wasMultiple: !!f.multiple };
                    D("invalid", d);
                    break;
                  case "textarea":
                    hb(d, f), D("invalid", d);
                }
                ub(c, f);
                e = null;
                for (var g in f) if (f.hasOwnProperty(g)) {
                  var h = f[g];
                  "children" === g ? "string" === typeof h ? d.textContent !== h && (true !== f.suppressHydrationWarning && Af(d.textContent, h, a), e = ["children", h]) : "number" === typeof h && d.textContent !== "" + h && (true !== f.suppressHydrationWarning && Af(
                    d.textContent,
                    h,
                    a
                  ), e = ["children", "" + h]) : ea.hasOwnProperty(g) && null != h && "onScroll" === g && D("scroll", d);
                }
                switch (c) {
                  case "input":
                    Va(d);
                    db(d, f, true);
                    break;
                  case "textarea":
                    Va(d);
                    jb(d);
                    break;
                  case "select":
                  case "option":
                    break;
                  default:
                    "function" === typeof f.onClick && (d.onclick = Bf);
                }
                d = e;
                b.updateQueue = d;
                null !== d && (b.flags |= 4);
              } else {
                g = 9 === e.nodeType ? e : e.ownerDocument;
                "http://www.w3.org/1999/xhtml" === a && (a = kb(c));
                "http://www.w3.org/1999/xhtml" === a ? "script" === c ? (a = g.createElement("div"), a.innerHTML = "<script><\/script>", a = a.removeChild(a.firstChild)) : "string" === typeof d.is ? a = g.createElement(c, { is: d.is }) : (a = g.createElement(c), "select" === c && (g = a, d.multiple ? g.multiple = true : d.size && (g.size = d.size))) : a = g.createElementNS(a, c);
                a[Of] = b;
                a[Pf] = d;
                zj(a, b, false, false);
                b.stateNode = a;
                a: {
                  g = vb(c, d);
                  switch (c) {
                    case "dialog":
                      D("cancel", a);
                      D("close", a);
                      e = d;
                      break;
                    case "iframe":
                    case "object":
                    case "embed":
                      D("load", a);
                      e = d;
                      break;
                    case "video":
                    case "audio":
                      for (e = 0; e < lf.length; e++) D(lf[e], a);
                      e = d;
                      break;
                    case "source":
                      D("error", a);
                      e = d;
                      break;
                    case "img":
                    case "image":
                    case "link":
                      D(
                        "error",
                        a
                      );
                      D("load", a);
                      e = d;
                      break;
                    case "details":
                      D("toggle", a);
                      e = d;
                      break;
                    case "input":
                      Za(a, d);
                      e = Ya(a, d);
                      D("invalid", a);
                      break;
                    case "option":
                      e = d;
                      break;
                    case "select":
                      a._wrapperState = { wasMultiple: !!d.multiple };
                      e = A({}, d, { value: void 0 });
                      D("invalid", a);
                      break;
                    case "textarea":
                      hb(a, d);
                      e = gb(a, d);
                      D("invalid", a);
                      break;
                    default:
                      e = d;
                  }
                  ub(c, e);
                  h = e;
                  for (f in h) if (h.hasOwnProperty(f)) {
                    var k = h[f];
                    "style" === f ? sb(a, k) : "dangerouslySetInnerHTML" === f ? (k = k ? k.__html : void 0, null != k && nb(a, k)) : "children" === f ? "string" === typeof k ? ("textarea" !== c || "" !== k) && ob(a, k) : "number" === typeof k && ob(a, "" + k) : "suppressContentEditableWarning" !== f && "suppressHydrationWarning" !== f && "autoFocus" !== f && (ea.hasOwnProperty(f) ? null != k && "onScroll" === f && D("scroll", a) : null != k && ta(a, f, k, g));
                  }
                  switch (c) {
                    case "input":
                      Va(a);
                      db(a, d, false);
                      break;
                    case "textarea":
                      Va(a);
                      jb(a);
                      break;
                    case "option":
                      null != d.value && a.setAttribute("value", "" + Sa(d.value));
                      break;
                    case "select":
                      a.multiple = !!d.multiple;
                      f = d.value;
                      null != f ? fb(a, !!d.multiple, f, false) : null != d.defaultValue && fb(
                        a,
                        !!d.multiple,
                        d.defaultValue,
                        true
                      );
                      break;
                    default:
                      "function" === typeof e.onClick && (a.onclick = Bf);
                  }
                  switch (c) {
                    case "button":
                    case "input":
                    case "select":
                    case "textarea":
                      d = !!d.autoFocus;
                      break a;
                    case "img":
                      d = true;
                      break a;
                    default:
                      d = false;
                  }
                }
                d && (b.flags |= 4);
              }
              null !== b.ref && (b.flags |= 512, b.flags |= 2097152);
            }
            S(b);
            return null;
          case 6:
            if (a && null != b.stateNode) Cj(a, b, a.memoizedProps, d);
            else {
              if ("string" !== typeof d && null === b.stateNode) throw Error(p(166));
              c = xh(wh.current);
              xh(uh.current);
              if (Gg(b)) {
                d = b.stateNode;
                c = b.memoizedProps;
                d[Of] = b;
                if (f = d.nodeValue !== c) {
                  if (a = xg, null !== a) switch (a.tag) {
                    case 3:
                      Af(d.nodeValue, c, 0 !== (a.mode & 1));
                      break;
                    case 5:
                      true !== a.memoizedProps.suppressHydrationWarning && Af(d.nodeValue, c, 0 !== (a.mode & 1));
                  }
                }
                f && (b.flags |= 4);
              } else d = (9 === c.nodeType ? c : c.ownerDocument).createTextNode(d), d[Of] = b, b.stateNode = d;
            }
            S(b);
            return null;
          case 13:
            E(L);
            d = b.memoizedState;
            if (null === a || null !== a.memoizedState && null !== a.memoizedState.dehydrated) {
              if (I && null !== yg && 0 !== (b.mode & 1) && 0 === (b.flags & 128)) Hg(), Ig(), b.flags |= 98560, f = false;
              else if (f = Gg(b), null !== d && null !== d.dehydrated) {
                if (null === a) {
                  if (!f) throw Error(p(318));
                  f = b.memoizedState;
                  f = null !== f ? f.dehydrated : null;
                  if (!f) throw Error(p(317));
                  f[Of] = b;
                } else Ig(), 0 === (b.flags & 128) && (b.memoizedState = null), b.flags |= 4;
                S(b);
                f = false;
              } else null !== zg && (Fj(zg), zg = null), f = true;
              if (!f) return b.flags & 65536 ? b : null;
            }
            if (0 !== (b.flags & 128)) return b.lanes = c, b;
            d = null !== d;
            d !== (null !== a && null !== a.memoizedState) && d && (b.child.flags |= 8192, 0 !== (b.mode & 1) && (null === a || 0 !== (L.current & 1) ? 0 === T && (T = 3) : tj()));
            null !== b.updateQueue && (b.flags |= 4);
            S(b);
            return null;
          case 4:
            return zh(), Aj(a, b), null === a && sf(b.stateNode.containerInfo), S(b), null;
          case 10:
            return ah(b.type._context), S(b), null;
          case 17:
            return Zf(b.type) && $f(), S(b), null;
          case 19:
            E(L);
            f = b.memoizedState;
            if (null === f) return S(b), null;
            d = 0 !== (b.flags & 128);
            g = f.rendering;
            if (null === g) if (d) Dj(f, false);
            else {
              if (0 !== T || null !== a && 0 !== (a.flags & 128)) for (a = b.child; null !== a; ) {
                g = Ch(a);
                if (null !== g) {
                  b.flags |= 128;
                  Dj(f, false);
                  d = g.updateQueue;
                  null !== d && (b.updateQueue = d, b.flags |= 4);
                  b.subtreeFlags = 0;
                  d = c;
                  for (c = b.child; null !== c; ) f = c, a = d, f.flags &= 14680066, g = f.alternate, null === g ? (f.childLanes = 0, f.lanes = a, f.child = null, f.subtreeFlags = 0, f.memoizedProps = null, f.memoizedState = null, f.updateQueue = null, f.dependencies = null, f.stateNode = null) : (f.childLanes = g.childLanes, f.lanes = g.lanes, f.child = g.child, f.subtreeFlags = 0, f.deletions = null, f.memoizedProps = g.memoizedProps, f.memoizedState = g.memoizedState, f.updateQueue = g.updateQueue, f.type = g.type, a = g.dependencies, f.dependencies = null === a ? null : { lanes: a.lanes, firstContext: a.firstContext }), c = c.sibling;
                  G(L, L.current & 1 | 2);
                  return b.child;
                }
                a = a.sibling;
              }
              null !== f.tail && B() > Gj && (b.flags |= 128, d = true, Dj(f, false), b.lanes = 4194304);
            }
            else {
              if (!d) if (a = Ch(g), null !== a) {
                if (b.flags |= 128, d = true, c = a.updateQueue, null !== c && (b.updateQueue = c, b.flags |= 4), Dj(f, true), null === f.tail && "hidden" === f.tailMode && !g.alternate && !I) return S(b), null;
              } else 2 * B() - f.renderingStartTime > Gj && 1073741824 !== c && (b.flags |= 128, d = true, Dj(f, false), b.lanes = 4194304);
              f.isBackwards ? (g.sibling = b.child, b.child = g) : (c = f.last, null !== c ? c.sibling = g : b.child = g, f.last = g);
            }
            if (null !== f.tail) return b = f.tail, f.rendering = b, f.tail = b.sibling, f.renderingStartTime = B(), b.sibling = null, c = L.current, G(L, d ? c & 1 | 2 : c & 1), b;
            S(b);
            return null;
          case 22:
          case 23:
            return Hj(), d = null !== b.memoizedState, null !== a && null !== a.memoizedState !== d && (b.flags |= 8192), d && 0 !== (b.mode & 1) ? 0 !== (fj & 1073741824) && (S(b), b.subtreeFlags & 6 && (b.flags |= 8192)) : S(b), null;
          case 24:
            return null;
          case 25:
            return null;
        }
        throw Error(p(156, b.tag));
      }
      function Ij(a, b) {
        wg(b);
        switch (b.tag) {
          case 1:
            return Zf(b.type) && $f(), a = b.flags, a & 65536 ? (b.flags = a & -65537 | 128, b) : null;
          case 3:
            return zh(), E(Wf), E(H), Eh(), a = b.flags, 0 !== (a & 65536) && 0 === (a & 128) ? (b.flags = a & -65537 | 128, b) : null;
          case 5:
            return Bh(b), null;
          case 13:
            E(L);
            a = b.memoizedState;
            if (null !== a && null !== a.dehydrated) {
              if (null === b.alternate) throw Error(p(340));
              Ig();
            }
            a = b.flags;
            return a & 65536 ? (b.flags = a & -65537 | 128, b) : null;
          case 19:
            return E(L), null;
          case 4:
            return zh(), null;
          case 10:
            return ah(b.type._context), null;
          case 22:
          case 23:
            return Hj(), null;
          case 24:
            return null;
          default:
            return null;
        }
      }
      var Jj = false;
      var U = false;
      var Kj = "function" === typeof WeakSet ? WeakSet : Set;
      var V = null;
      function Lj(a, b) {
        var c = a.ref;
        if (null !== c) if ("function" === typeof c) try {
          c(null);
        } catch (d) {
          W(a, b, d);
        }
        else c.current = null;
      }
      function Mj(a, b, c) {
        try {
          c();
        } catch (d) {
          W(a, b, d);
        }
      }
      var Nj = false;
      function Oj(a, b) {
        Cf = dd;
        a = Me();
        if (Ne(a)) {
          if ("selectionStart" in a) var c = { start: a.selectionStart, end: a.selectionEnd };
          else a: {
            c = (c = a.ownerDocument) && c.defaultView || window;
            var d = c.getSelection && c.getSelection();
            if (d && 0 !== d.rangeCount) {
              c = d.anchorNode;
              var e = d.anchorOffset, f = d.focusNode;
              d = d.focusOffset;
              try {
                c.nodeType, f.nodeType;
              } catch (F) {
                c = null;
                break a;
              }
              var g = 0, h = -1, k = -1, l = 0, m = 0, q = a, r2 = null;
              b: for (; ; ) {
                for (var y; ; ) {
                  q !== c || 0 !== e && 3 !== q.nodeType || (h = g + e);
                  q !== f || 0 !== d && 3 !== q.nodeType || (k = g + d);
                  3 === q.nodeType && (g += q.nodeValue.length);
                  if (null === (y = q.firstChild)) break;
                  r2 = q;
                  q = y;
                }
                for (; ; ) {
                  if (q === a) break b;
                  r2 === c && ++l === e && (h = g);
                  r2 === f && ++m === d && (k = g);
                  if (null !== (y = q.nextSibling)) break;
                  q = r2;
                  r2 = q.parentNode;
                }
                q = y;
              }
              c = -1 === h || -1 === k ? null : { start: h, end: k };
            } else c = null;
          }
          c = c || { start: 0, end: 0 };
        } else c = null;
        Df = { focusedElem: a, selectionRange: c };
        dd = false;
        for (V = b; null !== V; ) if (b = V, a = b.child, 0 !== (b.subtreeFlags & 1028) && null !== a) a.return = b, V = a;
        else for (; null !== V; ) {
          b = V;
          try {
            var n = b.alternate;
            if (0 !== (b.flags & 1024)) switch (b.tag) {
              case 0:
              case 11:
              case 15:
                break;
              case 1:
                if (null !== n) {
                  var t = n.memoizedProps, J = n.memoizedState, x = b.stateNode, w = x.getSnapshotBeforeUpdate(b.elementType === b.type ? t : Ci(b.type, t), J);
                  x.__reactInternalSnapshotBeforeUpdate = w;
                }
                break;
              case 3:
                var u = b.stateNode.containerInfo;
                1 === u.nodeType ? u.textContent = "" : 9 === u.nodeType && u.documentElement && u.removeChild(u.documentElement);
                break;
              case 5:
              case 6:
              case 4:
              case 17:
                break;
              default:
                throw Error(p(163));
            }
          } catch (F) {
            W(b, b.return, F);
          }
          a = b.sibling;
          if (null !== a) {
            a.return = b.return;
            V = a;
            break;
          }
          V = b.return;
        }
        n = Nj;
        Nj = false;
        return n;
      }
      function Pj(a, b, c) {
        var d = b.updateQueue;
        d = null !== d ? d.lastEffect : null;
        if (null !== d) {
          var e = d = d.next;
          do {
            if ((e.tag & a) === a) {
              var f = e.destroy;
              e.destroy = void 0;
              void 0 !== f && Mj(b, c, f);
            }
            e = e.next;
          } while (e !== d);
        }
      }
      function Qj(a, b) {
        b = b.updateQueue;
        b = null !== b ? b.lastEffect : null;
        if (null !== b) {
          var c = b = b.next;
          do {
            if ((c.tag & a) === a) {
              var d = c.create;
              c.destroy = d();
            }
            c = c.next;
          } while (c !== b);
        }
      }
      function Rj(a) {
        var b = a.ref;
        if (null !== b) {
          var c = a.stateNode;
          switch (a.tag) {
            case 5:
              a = c;
              break;
            default:
              a = c;
          }
          "function" === typeof b ? b(a) : b.current = a;
        }
      }
      function Sj(a) {
        var b = a.alternate;
        null !== b && (a.alternate = null, Sj(b));
        a.child = null;
        a.deletions = null;
        a.sibling = null;
        5 === a.tag && (b = a.stateNode, null !== b && (delete b[Of], delete b[Pf], delete b[of], delete b[Qf], delete b[Rf]));
        a.stateNode = null;
        a.return = null;
        a.dependencies = null;
        a.memoizedProps = null;
        a.memoizedState = null;
        a.pendingProps = null;
        a.stateNode = null;
        a.updateQueue = null;
      }
      function Tj(a) {
        return 5 === a.tag || 3 === a.tag || 4 === a.tag;
      }
      function Uj(a) {
        a: for (; ; ) {
          for (; null === a.sibling; ) {
            if (null === a.return || Tj(a.return)) return null;
            a = a.return;
          }
          a.sibling.return = a.return;
          for (a = a.sibling; 5 !== a.tag && 6 !== a.tag && 18 !== a.tag; ) {
            if (a.flags & 2) continue a;
            if (null === a.child || 4 === a.tag) continue a;
            else a.child.return = a, a = a.child;
          }
          if (!(a.flags & 2)) return a.stateNode;
        }
      }
      function Vj(a, b, c) {
        var d = a.tag;
        if (5 === d || 6 === d) a = a.stateNode, b ? 8 === c.nodeType ? c.parentNode.insertBefore(a, b) : c.insertBefore(a, b) : (8 === c.nodeType ? (b = c.parentNode, b.insertBefore(a, c)) : (b = c, b.appendChild(a)), c = c._reactRootContainer, null !== c && void 0 !== c || null !== b.onclick || (b.onclick = Bf));
        else if (4 !== d && (a = a.child, null !== a)) for (Vj(a, b, c), a = a.sibling; null !== a; ) Vj(a, b, c), a = a.sibling;
      }
      function Wj(a, b, c) {
        var d = a.tag;
        if (5 === d || 6 === d) a = a.stateNode, b ? c.insertBefore(a, b) : c.appendChild(a);
        else if (4 !== d && (a = a.child, null !== a)) for (Wj(a, b, c), a = a.sibling; null !== a; ) Wj(a, b, c), a = a.sibling;
      }
      var X = null;
      var Xj = false;
      function Yj(a, b, c) {
        for (c = c.child; null !== c; ) Zj(a, b, c), c = c.sibling;
      }
      function Zj(a, b, c) {
        if (lc && "function" === typeof lc.onCommitFiberUnmount) try {
          lc.onCommitFiberUnmount(kc, c);
        } catch (h) {
        }
        switch (c.tag) {
          case 5:
            U || Lj(c, b);
          case 6:
            var d = X, e = Xj;
            X = null;
            Yj(a, b, c);
            X = d;
            Xj = e;
            null !== X && (Xj ? (a = X, c = c.stateNode, 8 === a.nodeType ? a.parentNode.removeChild(c) : a.removeChild(c)) : X.removeChild(c.stateNode));
            break;
          case 18:
            null !== X && (Xj ? (a = X, c = c.stateNode, 8 === a.nodeType ? Kf(a.parentNode, c) : 1 === a.nodeType && Kf(a, c), bd(a)) : Kf(X, c.stateNode));
            break;
          case 4:
            d = X;
            e = Xj;
            X = c.stateNode.containerInfo;
            Xj = true;
            Yj(a, b, c);
            X = d;
            Xj = e;
            break;
          case 0:
          case 11:
          case 14:
          case 15:
            if (!U && (d = c.updateQueue, null !== d && (d = d.lastEffect, null !== d))) {
              e = d = d.next;
              do {
                var f = e, g = f.destroy;
                f = f.tag;
                void 0 !== g && (0 !== (f & 2) ? Mj(c, b, g) : 0 !== (f & 4) && Mj(c, b, g));
                e = e.next;
              } while (e !== d);
            }
            Yj(a, b, c);
            break;
          case 1:
            if (!U && (Lj(c, b), d = c.stateNode, "function" === typeof d.componentWillUnmount)) try {
              d.props = c.memoizedProps, d.state = c.memoizedState, d.componentWillUnmount();
            } catch (h) {
              W(c, b, h);
            }
            Yj(a, b, c);
            break;
          case 21:
            Yj(a, b, c);
            break;
          case 22:
            c.mode & 1 ? (U = (d = U) || null !== c.memoizedState, Yj(a, b, c), U = d) : Yj(a, b, c);
            break;
          default:
            Yj(a, b, c);
        }
      }
      function ak(a) {
        var b = a.updateQueue;
        if (null !== b) {
          a.updateQueue = null;
          var c = a.stateNode;
          null === c && (c = a.stateNode = new Kj());
          b.forEach(function(b2) {
            var d = bk.bind(null, a, b2);
            c.has(b2) || (c.add(b2), b2.then(d, d));
          });
        }
      }
      function ck(a, b) {
        var c = b.deletions;
        if (null !== c) for (var d = 0; d < c.length; d++) {
          var e = c[d];
          try {
            var f = a, g = b, h = g;
            a: for (; null !== h; ) {
              switch (h.tag) {
                case 5:
                  X = h.stateNode;
                  Xj = false;
                  break a;
                case 3:
                  X = h.stateNode.containerInfo;
                  Xj = true;
                  break a;
                case 4:
                  X = h.stateNode.containerInfo;
                  Xj = true;
                  break a;
              }
              h = h.return;
            }
            if (null === X) throw Error(p(160));
            Zj(f, g, e);
            X = null;
            Xj = false;
            var k = e.alternate;
            null !== k && (k.return = null);
            e.return = null;
          } catch (l) {
            W(e, b, l);
          }
        }
        if (b.subtreeFlags & 12854) for (b = b.child; null !== b; ) dk(b, a), b = b.sibling;
      }
      function dk(a, b) {
        var c = a.alternate, d = a.flags;
        switch (a.tag) {
          case 0:
          case 11:
          case 14:
          case 15:
            ck(b, a);
            ek(a);
            if (d & 4) {
              try {
                Pj(3, a, a.return), Qj(3, a);
              } catch (t) {
                W(a, a.return, t);
              }
              try {
                Pj(5, a, a.return);
              } catch (t) {
                W(a, a.return, t);
              }
            }
            break;
          case 1:
            ck(b, a);
            ek(a);
            d & 512 && null !== c && Lj(c, c.return);
            break;
          case 5:
            ck(b, a);
            ek(a);
            d & 512 && null !== c && Lj(c, c.return);
            if (a.flags & 32) {
              var e = a.stateNode;
              try {
                ob(e, "");
              } catch (t) {
                W(a, a.return, t);
              }
            }
            if (d & 4 && (e = a.stateNode, null != e)) {
              var f = a.memoizedProps, g = null !== c ? c.memoizedProps : f, h = a.type, k = a.updateQueue;
              a.updateQueue = null;
              if (null !== k) try {
                "input" === h && "radio" === f.type && null != f.name && ab(e, f);
                vb(h, g);
                var l = vb(h, f);
                for (g = 0; g < k.length; g += 2) {
                  var m = k[g], q = k[g + 1];
                  "style" === m ? sb(e, q) : "dangerouslySetInnerHTML" === m ? nb(e, q) : "children" === m ? ob(e, q) : ta(e, m, q, l);
                }
                switch (h) {
                  case "input":
                    bb(e, f);
                    break;
                  case "textarea":
                    ib(e, f);
                    break;
                  case "select":
                    var r2 = e._wrapperState.wasMultiple;
                    e._wrapperState.wasMultiple = !!f.multiple;
                    var y = f.value;
                    null != y ? fb(e, !!f.multiple, y, false) : r2 !== !!f.multiple && (null != f.defaultValue ? fb(
                      e,
                      !!f.multiple,
                      f.defaultValue,
                      true
                    ) : fb(e, !!f.multiple, f.multiple ? [] : "", false));
                }
                e[Pf] = f;
              } catch (t) {
                W(a, a.return, t);
              }
            }
            break;
          case 6:
            ck(b, a);
            ek(a);
            if (d & 4) {
              if (null === a.stateNode) throw Error(p(162));
              e = a.stateNode;
              f = a.memoizedProps;
              try {
                e.nodeValue = f;
              } catch (t) {
                W(a, a.return, t);
              }
            }
            break;
          case 3:
            ck(b, a);
            ek(a);
            if (d & 4 && null !== c && c.memoizedState.isDehydrated) try {
              bd(b.containerInfo);
            } catch (t) {
              W(a, a.return, t);
            }
            break;
          case 4:
            ck(b, a);
            ek(a);
            break;
          case 13:
            ck(b, a);
            ek(a);
            e = a.child;
            e.flags & 8192 && (f = null !== e.memoizedState, e.stateNode.isHidden = f, !f || null !== e.alternate && null !== e.alternate.memoizedState || (fk = B()));
            d & 4 && ak(a);
            break;
          case 22:
            m = null !== c && null !== c.memoizedState;
            a.mode & 1 ? (U = (l = U) || m, ck(b, a), U = l) : ck(b, a);
            ek(a);
            if (d & 8192) {
              l = null !== a.memoizedState;
              if ((a.stateNode.isHidden = l) && !m && 0 !== (a.mode & 1)) for (V = a, m = a.child; null !== m; ) {
                for (q = V = m; null !== V; ) {
                  r2 = V;
                  y = r2.child;
                  switch (r2.tag) {
                    case 0:
                    case 11:
                    case 14:
                    case 15:
                      Pj(4, r2, r2.return);
                      break;
                    case 1:
                      Lj(r2, r2.return);
                      var n = r2.stateNode;
                      if ("function" === typeof n.componentWillUnmount) {
                        d = r2;
                        c = r2.return;
                        try {
                          b = d, n.props = b.memoizedProps, n.state = b.memoizedState, n.componentWillUnmount();
                        } catch (t) {
                          W(d, c, t);
                        }
                      }
                      break;
                    case 5:
                      Lj(r2, r2.return);
                      break;
                    case 22:
                      if (null !== r2.memoizedState) {
                        gk(q);
                        continue;
                      }
                  }
                  null !== y ? (y.return = r2, V = y) : gk(q);
                }
                m = m.sibling;
              }
              a: for (m = null, q = a; ; ) {
                if (5 === q.tag) {
                  if (null === m) {
                    m = q;
                    try {
                      e = q.stateNode, l ? (f = e.style, "function" === typeof f.setProperty ? f.setProperty("display", "none", "important") : f.display = "none") : (h = q.stateNode, k = q.memoizedProps.style, g = void 0 !== k && null !== k && k.hasOwnProperty("display") ? k.display : null, h.style.display = rb("display", g));
                    } catch (t) {
                      W(a, a.return, t);
                    }
                  }
                } else if (6 === q.tag) {
                  if (null === m) try {
                    q.stateNode.nodeValue = l ? "" : q.memoizedProps;
                  } catch (t) {
                    W(a, a.return, t);
                  }
                } else if ((22 !== q.tag && 23 !== q.tag || null === q.memoizedState || q === a) && null !== q.child) {
                  q.child.return = q;
                  q = q.child;
                  continue;
                }
                if (q === a) break a;
                for (; null === q.sibling; ) {
                  if (null === q.return || q.return === a) break a;
                  m === q && (m = null);
                  q = q.return;
                }
                m === q && (m = null);
                q.sibling.return = q.return;
                q = q.sibling;
              }
            }
            break;
          case 19:
            ck(b, a);
            ek(a);
            d & 4 && ak(a);
            break;
          case 21:
            break;
          default:
            ck(
              b,
              a
            ), ek(a);
        }
      }
      function ek(a) {
        var b = a.flags;
        if (b & 2) {
          try {
            a: {
              for (var c = a.return; null !== c; ) {
                if (Tj(c)) {
                  var d = c;
                  break a;
                }
                c = c.return;
              }
              throw Error(p(160));
            }
            switch (d.tag) {
              case 5:
                var e = d.stateNode;
                d.flags & 32 && (ob(e, ""), d.flags &= -33);
                var f = Uj(a);
                Wj(a, f, e);
                break;
              case 3:
              case 4:
                var g = d.stateNode.containerInfo, h = Uj(a);
                Vj(a, h, g);
                break;
              default:
                throw Error(p(161));
            }
          } catch (k) {
            W(a, a.return, k);
          }
          a.flags &= -3;
        }
        b & 4096 && (a.flags &= -4097);
      }
      function hk(a, b, c) {
        V = a;
        ik(a, b, c);
      }
      function ik(a, b, c) {
        for (var d = 0 !== (a.mode & 1); null !== V; ) {
          var e = V, f = e.child;
          if (22 === e.tag && d) {
            var g = null !== e.memoizedState || Jj;
            if (!g) {
              var h = e.alternate, k = null !== h && null !== h.memoizedState || U;
              h = Jj;
              var l = U;
              Jj = g;
              if ((U = k) && !l) for (V = e; null !== V; ) g = V, k = g.child, 22 === g.tag && null !== g.memoizedState ? jk(e) : null !== k ? (k.return = g, V = k) : jk(e);
              for (; null !== f; ) V = f, ik(f, b, c), f = f.sibling;
              V = e;
              Jj = h;
              U = l;
            }
            kk(a, b, c);
          } else 0 !== (e.subtreeFlags & 8772) && null !== f ? (f.return = e, V = f) : kk(a, b, c);
        }
      }
      function kk(a) {
        for (; null !== V; ) {
          var b = V;
          if (0 !== (b.flags & 8772)) {
            var c = b.alternate;
            try {
              if (0 !== (b.flags & 8772)) switch (b.tag) {
                case 0:
                case 11:
                case 15:
                  U || Qj(5, b);
                  break;
                case 1:
                  var d = b.stateNode;
                  if (b.flags & 4 && !U) if (null === c) d.componentDidMount();
                  else {
                    var e = b.elementType === b.type ? c.memoizedProps : Ci(b.type, c.memoizedProps);
                    d.componentDidUpdate(e, c.memoizedState, d.__reactInternalSnapshotBeforeUpdate);
                  }
                  var f = b.updateQueue;
                  null !== f && sh(b, f, d);
                  break;
                case 3:
                  var g = b.updateQueue;
                  if (null !== g) {
                    c = null;
                    if (null !== b.child) switch (b.child.tag) {
                      case 5:
                        c = b.child.stateNode;
                        break;
                      case 1:
                        c = b.child.stateNode;
                    }
                    sh(b, g, c);
                  }
                  break;
                case 5:
                  var h = b.stateNode;
                  if (null === c && b.flags & 4) {
                    c = h;
                    var k = b.memoizedProps;
                    switch (b.type) {
                      case "button":
                      case "input":
                      case "select":
                      case "textarea":
                        k.autoFocus && c.focus();
                        break;
                      case "img":
                        k.src && (c.src = k.src);
                    }
                  }
                  break;
                case 6:
                  break;
                case 4:
                  break;
                case 12:
                  break;
                case 13:
                  if (null === b.memoizedState) {
                    var l = b.alternate;
                    if (null !== l) {
                      var m = l.memoizedState;
                      if (null !== m) {
                        var q = m.dehydrated;
                        null !== q && bd(q);
                      }
                    }
                  }
                  break;
                case 19:
                case 17:
                case 21:
                case 22:
                case 23:
                case 25:
                  break;
                default:
                  throw Error(p(163));
              }
              U || b.flags & 512 && Rj(b);
            } catch (r2) {
              W(b, b.return, r2);
            }
          }
          if (b === a) {
            V = null;
            break;
          }
          c = b.sibling;
          if (null !== c) {
            c.return = b.return;
            V = c;
            break;
          }
          V = b.return;
        }
      }
      function gk(a) {
        for (; null !== V; ) {
          var b = V;
          if (b === a) {
            V = null;
            break;
          }
          var c = b.sibling;
          if (null !== c) {
            c.return = b.return;
            V = c;
            break;
          }
          V = b.return;
        }
      }
      function jk(a) {
        for (; null !== V; ) {
          var b = V;
          try {
            switch (b.tag) {
              case 0:
              case 11:
              case 15:
                var c = b.return;
                try {
                  Qj(4, b);
                } catch (k) {
                  W(b, c, k);
                }
                break;
              case 1:
                var d = b.stateNode;
                if ("function" === typeof d.componentDidMount) {
                  var e = b.return;
                  try {
                    d.componentDidMount();
                  } catch (k) {
                    W(b, e, k);
                  }
                }
                var f = b.return;
                try {
                  Rj(b);
                } catch (k) {
                  W(b, f, k);
                }
                break;
              case 5:
                var g = b.return;
                try {
                  Rj(b);
                } catch (k) {
                  W(b, g, k);
                }
            }
          } catch (k) {
            W(b, b.return, k);
          }
          if (b === a) {
            V = null;
            break;
          }
          var h = b.sibling;
          if (null !== h) {
            h.return = b.return;
            V = h;
            break;
          }
          V = b.return;
        }
      }
      var lk = Math.ceil;
      var mk = ua.ReactCurrentDispatcher;
      var nk = ua.ReactCurrentOwner;
      var ok = ua.ReactCurrentBatchConfig;
      var K = 0;
      var Q = null;
      var Y = null;
      var Z = 0;
      var fj = 0;
      var ej = Uf(0);
      var T = 0;
      var pk = null;
      var rh = 0;
      var qk = 0;
      var rk = 0;
      var sk = null;
      var tk = null;
      var fk = 0;
      var Gj = Infinity;
      var uk = null;
      var Oi = false;
      var Pi = null;
      var Ri = null;
      var vk = false;
      var wk = null;
      var xk = 0;
      var yk = 0;
      var zk = null;
      var Ak = -1;
      var Bk = 0;
      function R() {
        return 0 !== (K & 6) ? B() : -1 !== Ak ? Ak : Ak = B();
      }
      function yi(a) {
        if (0 === (a.mode & 1)) return 1;
        if (0 !== (K & 2) && 0 !== Z) return Z & -Z;
        if (null !== Kg.transition) return 0 === Bk && (Bk = yc()), Bk;
        a = C;
        if (0 !== a) return a;
        a = window.event;
        a = void 0 === a ? 16 : jd(a.type);
        return a;
      }
      function gi(a, b, c, d) {
        if (50 < yk) throw yk = 0, zk = null, Error(p(185));
        Ac(a, c, d);
        if (0 === (K & 2) || a !== Q) a === Q && (0 === (K & 2) && (qk |= c), 4 === T && Ck(a, Z)), Dk(a, d), 1 === c && 0 === K && 0 === (b.mode & 1) && (Gj = B() + 500, fg && jg());
      }
      function Dk(a, b) {
        var c = a.callbackNode;
        wc(a, b);
        var d = uc(a, a === Q ? Z : 0);
        if (0 === d) null !== c && bc(c), a.callbackNode = null, a.callbackPriority = 0;
        else if (b = d & -d, a.callbackPriority !== b) {
          null != c && bc(c);
          if (1 === b) 0 === a.tag ? ig(Ek.bind(null, a)) : hg(Ek.bind(null, a)), Jf(function() {
            0 === (K & 6) && jg();
          }), c = null;
          else {
            switch (Dc(d)) {
              case 1:
                c = fc;
                break;
              case 4:
                c = gc;
                break;
              case 16:
                c = hc;
                break;
              case 536870912:
                c = jc;
                break;
              default:
                c = hc;
            }
            c = Fk(c, Gk.bind(null, a));
          }
          a.callbackPriority = b;
          a.callbackNode = c;
        }
      }
      function Gk(a, b) {
        Ak = -1;
        Bk = 0;
        if (0 !== (K & 6)) throw Error(p(327));
        var c = a.callbackNode;
        if (Hk() && a.callbackNode !== c) return null;
        var d = uc(a, a === Q ? Z : 0);
        if (0 === d) return null;
        if (0 !== (d & 30) || 0 !== (d & a.expiredLanes) || b) b = Ik(a, d);
        else {
          b = d;
          var e = K;
          K |= 2;
          var f = Jk();
          if (Q !== a || Z !== b) uk = null, Gj = B() + 500, Kk(a, b);
          do
            try {
              Lk();
              break;
            } catch (h) {
              Mk(a, h);
            }
          while (1);
          $g();
          mk.current = f;
          K = e;
          null !== Y ? b = 0 : (Q = null, Z = 0, b = T);
        }
        if (0 !== b) {
          2 === b && (e = xc(a), 0 !== e && (d = e, b = Nk(a, e)));
          if (1 === b) throw c = pk, Kk(a, 0), Ck(a, d), Dk(a, B()), c;
          if (6 === b) Ck(a, d);
          else {
            e = a.current.alternate;
            if (0 === (d & 30) && !Ok(e) && (b = Ik(a, d), 2 === b && (f = xc(a), 0 !== f && (d = f, b = Nk(a, f))), 1 === b)) throw c = pk, Kk(a, 0), Ck(a, d), Dk(a, B()), c;
            a.finishedWork = e;
            a.finishedLanes = d;
            switch (b) {
              case 0:
              case 1:
                throw Error(p(345));
              case 2:
                Pk(a, tk, uk);
                break;
              case 3:
                Ck(a, d);
                if ((d & 130023424) === d && (b = fk + 500 - B(), 10 < b)) {
                  if (0 !== uc(a, 0)) break;
                  e = a.suspendedLanes;
                  if ((e & d) !== d) {
                    R();
                    a.pingedLanes |= a.suspendedLanes & e;
                    break;
                  }
                  a.timeoutHandle = Ff(Pk.bind(null, a, tk, uk), b);
                  break;
                }
                Pk(a, tk, uk);
                break;
              case 4:
                Ck(a, d);
                if ((d & 4194240) === d) break;
                b = a.eventTimes;
                for (e = -1; 0 < d; ) {
                  var g = 31 - oc(d);
                  f = 1 << g;
                  g = b[g];
                  g > e && (e = g);
                  d &= ~f;
                }
                d = e;
                d = B() - d;
                d = (120 > d ? 120 : 480 > d ? 480 : 1080 > d ? 1080 : 1920 > d ? 1920 : 3e3 > d ? 3e3 : 4320 > d ? 4320 : 1960 * lk(d / 1960)) - d;
                if (10 < d) {
                  a.timeoutHandle = Ff(Pk.bind(null, a, tk, uk), d);
                  break;
                }
                Pk(a, tk, uk);
                break;
              case 5:
                Pk(a, tk, uk);
                break;
              default:
                throw Error(p(329));
            }
          }
        }
        Dk(a, B());
        return a.callbackNode === c ? Gk.bind(null, a) : null;
      }
      function Nk(a, b) {
        var c = sk;
        a.current.memoizedState.isDehydrated && (Kk(a, b).flags |= 256);
        a = Ik(a, b);
        2 !== a && (b = tk, tk = c, null !== b && Fj(b));
        return a;
      }
      function Fj(a) {
        null === tk ? tk = a : tk.push.apply(tk, a);
      }
      function Ok(a) {
        for (var b = a; ; ) {
          if (b.flags & 16384) {
            var c = b.updateQueue;
            if (null !== c && (c = c.stores, null !== c)) for (var d = 0; d < c.length; d++) {
              var e = c[d], f = e.getSnapshot;
              e = e.value;
              try {
                if (!He(f(), e)) return false;
              } catch (g) {
                return false;
              }
            }
          }
          c = b.child;
          if (b.subtreeFlags & 16384 && null !== c) c.return = b, b = c;
          else {
            if (b === a) break;
            for (; null === b.sibling; ) {
              if (null === b.return || b.return === a) return true;
              b = b.return;
            }
            b.sibling.return = b.return;
            b = b.sibling;
          }
        }
        return true;
      }
      function Ck(a, b) {
        b &= ~rk;
        b &= ~qk;
        a.suspendedLanes |= b;
        a.pingedLanes &= ~b;
        for (a = a.expirationTimes; 0 < b; ) {
          var c = 31 - oc(b), d = 1 << c;
          a[c] = -1;
          b &= ~d;
        }
      }
      function Ek(a) {
        if (0 !== (K & 6)) throw Error(p(327));
        Hk();
        var b = uc(a, 0);
        if (0 === (b & 1)) return Dk(a, B()), null;
        var c = Ik(a, b);
        if (0 !== a.tag && 2 === c) {
          var d = xc(a);
          0 !== d && (b = d, c = Nk(a, d));
        }
        if (1 === c) throw c = pk, Kk(a, 0), Ck(a, b), Dk(a, B()), c;
        if (6 === c) throw Error(p(345));
        a.finishedWork = a.current.alternate;
        a.finishedLanes = b;
        Pk(a, tk, uk);
        Dk(a, B());
        return null;
      }
      function Qk(a, b) {
        var c = K;
        K |= 1;
        try {
          return a(b);
        } finally {
          K = c, 0 === K && (Gj = B() + 500, fg && jg());
        }
      }
      function Rk(a) {
        null !== wk && 0 === wk.tag && 0 === (K & 6) && Hk();
        var b = K;
        K |= 1;
        var c = ok.transition, d = C;
        try {
          if (ok.transition = null, C = 1, a) return a();
        } finally {
          C = d, ok.transition = c, K = b, 0 === (K & 6) && jg();
        }
      }
      function Hj() {
        fj = ej.current;
        E(ej);
      }
      function Kk(a, b) {
        a.finishedWork = null;
        a.finishedLanes = 0;
        var c = a.timeoutHandle;
        -1 !== c && (a.timeoutHandle = -1, Gf(c));
        if (null !== Y) for (c = Y.return; null !== c; ) {
          var d = c;
          wg(d);
          switch (d.tag) {
            case 1:
              d = d.type.childContextTypes;
              null !== d && void 0 !== d && $f();
              break;
            case 3:
              zh();
              E(Wf);
              E(H);
              Eh();
              break;
            case 5:
              Bh(d);
              break;
            case 4:
              zh();
              break;
            case 13:
              E(L);
              break;
            case 19:
              E(L);
              break;
            case 10:
              ah(d.type._context);
              break;
            case 22:
            case 23:
              Hj();
          }
          c = c.return;
        }
        Q = a;
        Y = a = Pg(a.current, null);
        Z = fj = b;
        T = 0;
        pk = null;
        rk = qk = rh = 0;
        tk = sk = null;
        if (null !== fh) {
          for (b = 0; b < fh.length; b++) if (c = fh[b], d = c.interleaved, null !== d) {
            c.interleaved = null;
            var e = d.next, f = c.pending;
            if (null !== f) {
              var g = f.next;
              f.next = e;
              d.next = g;
            }
            c.pending = d;
          }
          fh = null;
        }
        return a;
      }
      function Mk(a, b) {
        do {
          var c = Y;
          try {
            $g();
            Fh.current = Rh;
            if (Ih) {
              for (var d = M.memoizedState; null !== d; ) {
                var e = d.queue;
                null !== e && (e.pending = null);
                d = d.next;
              }
              Ih = false;
            }
            Hh = 0;
            O = N = M = null;
            Jh = false;
            Kh = 0;
            nk.current = null;
            if (null === c || null === c.return) {
              T = 1;
              pk = b;
              Y = null;
              break;
            }
            a: {
              var f = a, g = c.return, h = c, k = b;
              b = Z;
              h.flags |= 32768;
              if (null !== k && "object" === typeof k && "function" === typeof k.then) {
                var l = k, m = h, q = m.tag;
                if (0 === (m.mode & 1) && (0 === q || 11 === q || 15 === q)) {
                  var r2 = m.alternate;
                  r2 ? (m.updateQueue = r2.updateQueue, m.memoizedState = r2.memoizedState, m.lanes = r2.lanes) : (m.updateQueue = null, m.memoizedState = null);
                }
                var y = Ui(g);
                if (null !== y) {
                  y.flags &= -257;
                  Vi(y, g, h, f, b);
                  y.mode & 1 && Si(f, l, b);
                  b = y;
                  k = l;
                  var n = b.updateQueue;
                  if (null === n) {
                    var t = /* @__PURE__ */ new Set();
                    t.add(k);
                    b.updateQueue = t;
                  } else n.add(k);
                  break a;
                } else {
                  if (0 === (b & 1)) {
                    Si(f, l, b);
                    tj();
                    break a;
                  }
                  k = Error(p(426));
                }
              } else if (I && h.mode & 1) {
                var J = Ui(g);
                if (null !== J) {
                  0 === (J.flags & 65536) && (J.flags |= 256);
                  Vi(J, g, h, f, b);
                  Jg(Ji(k, h));
                  break a;
                }
              }
              f = k = Ji(k, h);
              4 !== T && (T = 2);
              null === sk ? sk = [f] : sk.push(f);
              f = g;
              do {
                switch (f.tag) {
                  case 3:
                    f.flags |= 65536;
                    b &= -b;
                    f.lanes |= b;
                    var x = Ni(f, k, b);
                    ph(f, x);
                    break a;
                  case 1:
                    h = k;
                    var w = f.type, u = f.stateNode;
                    if (0 === (f.flags & 128) && ("function" === typeof w.getDerivedStateFromError || null !== u && "function" === typeof u.componentDidCatch && (null === Ri || !Ri.has(u)))) {
                      f.flags |= 65536;
                      b &= -b;
                      f.lanes |= b;
                      var F = Qi(f, h, b);
                      ph(f, F);
                      break a;
                    }
                }
                f = f.return;
              } while (null !== f);
            }
            Sk(c);
          } catch (na) {
            b = na;
            Y === c && null !== c && (Y = c = c.return);
            continue;
          }
          break;
        } while (1);
      }
      function Jk() {
        var a = mk.current;
        mk.current = Rh;
        return null === a ? Rh : a;
      }
      function tj() {
        if (0 === T || 3 === T || 2 === T) T = 4;
        null === Q || 0 === (rh & 268435455) && 0 === (qk & 268435455) || Ck(Q, Z);
      }
      function Ik(a, b) {
        var c = K;
        K |= 2;
        var d = Jk();
        if (Q !== a || Z !== b) uk = null, Kk(a, b);
        do
          try {
            Tk();
            break;
          } catch (e) {
            Mk(a, e);
          }
        while (1);
        $g();
        K = c;
        mk.current = d;
        if (null !== Y) throw Error(p(261));
        Q = null;
        Z = 0;
        return T;
      }
      function Tk() {
        for (; null !== Y; ) Uk(Y);
      }
      function Lk() {
        for (; null !== Y && !cc(); ) Uk(Y);
      }
      function Uk(a) {
        var b = Vk(a.alternate, a, fj);
        a.memoizedProps = a.pendingProps;
        null === b ? Sk(a) : Y = b;
        nk.current = null;
      }
      function Sk(a) {
        var b = a;
        do {
          var c = b.alternate;
          a = b.return;
          if (0 === (b.flags & 32768)) {
            if (c = Ej(c, b, fj), null !== c) {
              Y = c;
              return;
            }
          } else {
            c = Ij(c, b);
            if (null !== c) {
              c.flags &= 32767;
              Y = c;
              return;
            }
            if (null !== a) a.flags |= 32768, a.subtreeFlags = 0, a.deletions = null;
            else {
              T = 6;
              Y = null;
              return;
            }
          }
          b = b.sibling;
          if (null !== b) {
            Y = b;
            return;
          }
          Y = b = a;
        } while (null !== b);
        0 === T && (T = 5);
      }
      function Pk(a, b, c) {
        var d = C, e = ok.transition;
        try {
          ok.transition = null, C = 1, Wk(a, b, c, d);
        } finally {
          ok.transition = e, C = d;
        }
        return null;
      }
      function Wk(a, b, c, d) {
        do
          Hk();
        while (null !== wk);
        if (0 !== (K & 6)) throw Error(p(327));
        c = a.finishedWork;
        var e = a.finishedLanes;
        if (null === c) return null;
        a.finishedWork = null;
        a.finishedLanes = 0;
        if (c === a.current) throw Error(p(177));
        a.callbackNode = null;
        a.callbackPriority = 0;
        var f = c.lanes | c.childLanes;
        Bc(a, f);
        a === Q && (Y = Q = null, Z = 0);
        0 === (c.subtreeFlags & 2064) && 0 === (c.flags & 2064) || vk || (vk = true, Fk(hc, function() {
          Hk();
          return null;
        }));
        f = 0 !== (c.flags & 15990);
        if (0 !== (c.subtreeFlags & 15990) || f) {
          f = ok.transition;
          ok.transition = null;
          var g = C;
          C = 1;
          var h = K;
          K |= 4;
          nk.current = null;
          Oj(a, c);
          dk(c, a);
          Oe(Df);
          dd = !!Cf;
          Df = Cf = null;
          a.current = c;
          hk(c, a, e);
          dc();
          K = h;
          C = g;
          ok.transition = f;
        } else a.current = c;
        vk && (vk = false, wk = a, xk = e);
        f = a.pendingLanes;
        0 === f && (Ri = null);
        mc(c.stateNode, d);
        Dk(a, B());
        if (null !== b) for (d = a.onRecoverableError, c = 0; c < b.length; c++) e = b[c], d(e.value, { componentStack: e.stack, digest: e.digest });
        if (Oi) throw Oi = false, a = Pi, Pi = null, a;
        0 !== (xk & 1) && 0 !== a.tag && Hk();
        f = a.pendingLanes;
        0 !== (f & 1) ? a === zk ? yk++ : (yk = 0, zk = a) : yk = 0;
        jg();
        return null;
      }
      function Hk() {
        if (null !== wk) {
          var a = Dc(xk), b = ok.transition, c = C;
          try {
            ok.transition = null;
            C = 16 > a ? 16 : a;
            if (null === wk) var d = false;
            else {
              a = wk;
              wk = null;
              xk = 0;
              if (0 !== (K & 6)) throw Error(p(331));
              var e = K;
              K |= 4;
              for (V = a.current; null !== V; ) {
                var f = V, g = f.child;
                if (0 !== (V.flags & 16)) {
                  var h = f.deletions;
                  if (null !== h) {
                    for (var k = 0; k < h.length; k++) {
                      var l = h[k];
                      for (V = l; null !== V; ) {
                        var m = V;
                        switch (m.tag) {
                          case 0:
                          case 11:
                          case 15:
                            Pj(8, m, f);
                        }
                        var q = m.child;
                        if (null !== q) q.return = m, V = q;
                        else for (; null !== V; ) {
                          m = V;
                          var r2 = m.sibling, y = m.return;
                          Sj(m);
                          if (m === l) {
                            V = null;
                            break;
                          }
                          if (null !== r2) {
                            r2.return = y;
                            V = r2;
                            break;
                          }
                          V = y;
                        }
                      }
                    }
                    var n = f.alternate;
                    if (null !== n) {
                      var t = n.child;
                      if (null !== t) {
                        n.child = null;
                        do {
                          var J = t.sibling;
                          t.sibling = null;
                          t = J;
                        } while (null !== t);
                      }
                    }
                    V = f;
                  }
                }
                if (0 !== (f.subtreeFlags & 2064) && null !== g) g.return = f, V = g;
                else b: for (; null !== V; ) {
                  f = V;
                  if (0 !== (f.flags & 2048)) switch (f.tag) {
                    case 0:
                    case 11:
                    case 15:
                      Pj(9, f, f.return);
                  }
                  var x = f.sibling;
                  if (null !== x) {
                    x.return = f.return;
                    V = x;
                    break b;
                  }
                  V = f.return;
                }
              }
              var w = a.current;
              for (V = w; null !== V; ) {
                g = V;
                var u = g.child;
                if (0 !== (g.subtreeFlags & 2064) && null !== u) u.return = g, V = u;
                else b: for (g = w; null !== V; ) {
                  h = V;
                  if (0 !== (h.flags & 2048)) try {
                    switch (h.tag) {
                      case 0:
                      case 11:
                      case 15:
                        Qj(9, h);
                    }
                  } catch (na) {
                    W(h, h.return, na);
                  }
                  if (h === g) {
                    V = null;
                    break b;
                  }
                  var F = h.sibling;
                  if (null !== F) {
                    F.return = h.return;
                    V = F;
                    break b;
                  }
                  V = h.return;
                }
              }
              K = e;
              jg();
              if (lc && "function" === typeof lc.onPostCommitFiberRoot) try {
                lc.onPostCommitFiberRoot(kc, a);
              } catch (na) {
              }
              d = true;
            }
            return d;
          } finally {
            C = c, ok.transition = b;
          }
        }
        return false;
      }
      function Xk(a, b, c) {
        b = Ji(c, b);
        b = Ni(a, b, 1);
        a = nh(a, b, 1);
        b = R();
        null !== a && (Ac(a, 1, b), Dk(a, b));
      }
      function W(a, b, c) {
        if (3 === a.tag) Xk(a, a, c);
        else for (; null !== b; ) {
          if (3 === b.tag) {
            Xk(b, a, c);
            break;
          } else if (1 === b.tag) {
            var d = b.stateNode;
            if ("function" === typeof b.type.getDerivedStateFromError || "function" === typeof d.componentDidCatch && (null === Ri || !Ri.has(d))) {
              a = Ji(c, a);
              a = Qi(b, a, 1);
              b = nh(b, a, 1);
              a = R();
              null !== b && (Ac(b, 1, a), Dk(b, a));
              break;
            }
          }
          b = b.return;
        }
      }
      function Ti(a, b, c) {
        var d = a.pingCache;
        null !== d && d.delete(b);
        b = R();
        a.pingedLanes |= a.suspendedLanes & c;
        Q === a && (Z & c) === c && (4 === T || 3 === T && (Z & 130023424) === Z && 500 > B() - fk ? Kk(a, 0) : rk |= c);
        Dk(a, b);
      }
      function Yk(a, b) {
        0 === b && (0 === (a.mode & 1) ? b = 1 : (b = sc, sc <<= 1, 0 === (sc & 130023424) && (sc = 4194304)));
        var c = R();
        a = ih(a, b);
        null !== a && (Ac(a, b, c), Dk(a, c));
      }
      function uj(a) {
        var b = a.memoizedState, c = 0;
        null !== b && (c = b.retryLane);
        Yk(a, c);
      }
      function bk(a, b) {
        var c = 0;
        switch (a.tag) {
          case 13:
            var d = a.stateNode;
            var e = a.memoizedState;
            null !== e && (c = e.retryLane);
            break;
          case 19:
            d = a.stateNode;
            break;
          default:
            throw Error(p(314));
        }
        null !== d && d.delete(b);
        Yk(a, c);
      }
      var Vk;
      Vk = function(a, b, c) {
        if (null !== a) if (a.memoizedProps !== b.pendingProps || Wf.current) dh = true;
        else {
          if (0 === (a.lanes & c) && 0 === (b.flags & 128)) return dh = false, yj(a, b, c);
          dh = 0 !== (a.flags & 131072) ? true : false;
        }
        else dh = false, I && 0 !== (b.flags & 1048576) && ug(b, ng, b.index);
        b.lanes = 0;
        switch (b.tag) {
          case 2:
            var d = b.type;
            ij(a, b);
            a = b.pendingProps;
            var e = Yf(b, H.current);
            ch(b, c);
            e = Nh(null, b, d, a, e, c);
            var f = Sh();
            b.flags |= 1;
            "object" === typeof e && null !== e && "function" === typeof e.render && void 0 === e.$$typeof ? (b.tag = 1, b.memoizedState = null, b.updateQueue = null, Zf(d) ? (f = true, cg(b)) : f = false, b.memoizedState = null !== e.state && void 0 !== e.state ? e.state : null, kh(b), e.updater = Ei, b.stateNode = e, e._reactInternals = b, Ii(b, d, a, c), b = jj(null, b, d, true, f, c)) : (b.tag = 0, I && f && vg(b), Xi(null, b, e, c), b = b.child);
            return b;
          case 16:
            d = b.elementType;
            a: {
              ij(a, b);
              a = b.pendingProps;
              e = d._init;
              d = e(d._payload);
              b.type = d;
              e = b.tag = Zk(d);
              a = Ci(d, a);
              switch (e) {
                case 0:
                  b = cj(null, b, d, a, c);
                  break a;
                case 1:
                  b = hj(null, b, d, a, c);
                  break a;
                case 11:
                  b = Yi(null, b, d, a, c);
                  break a;
                case 14:
                  b = $i(null, b, d, Ci(d.type, a), c);
                  break a;
              }
              throw Error(p(
                306,
                d,
                ""
              ));
            }
            return b;
          case 0:
            return d = b.type, e = b.pendingProps, e = b.elementType === d ? e : Ci(d, e), cj(a, b, d, e, c);
          case 1:
            return d = b.type, e = b.pendingProps, e = b.elementType === d ? e : Ci(d, e), hj(a, b, d, e, c);
          case 3:
            a: {
              kj(b);
              if (null === a) throw Error(p(387));
              d = b.pendingProps;
              f = b.memoizedState;
              e = f.element;
              lh(a, b);
              qh(b, d, null, c);
              var g = b.memoizedState;
              d = g.element;
              if (f.isDehydrated) if (f = { element: d, isDehydrated: false, cache: g.cache, pendingSuspenseBoundaries: g.pendingSuspenseBoundaries, transitions: g.transitions }, b.updateQueue.baseState = f, b.memoizedState = f, b.flags & 256) {
                e = Ji(Error(p(423)), b);
                b = lj(a, b, d, c, e);
                break a;
              } else if (d !== e) {
                e = Ji(Error(p(424)), b);
                b = lj(a, b, d, c, e);
                break a;
              } else for (yg = Lf(b.stateNode.containerInfo.firstChild), xg = b, I = true, zg = null, c = Vg(b, null, d, c), b.child = c; c; ) c.flags = c.flags & -3 | 4096, c = c.sibling;
              else {
                Ig();
                if (d === e) {
                  b = Zi(a, b, c);
                  break a;
                }
                Xi(a, b, d, c);
              }
              b = b.child;
            }
            return b;
          case 5:
            return Ah(b), null === a && Eg(b), d = b.type, e = b.pendingProps, f = null !== a ? a.memoizedProps : null, g = e.children, Ef(d, e) ? g = null : null !== f && Ef(d, f) && (b.flags |= 32), gj(a, b), Xi(a, b, g, c), b.child;
          case 6:
            return null === a && Eg(b), null;
          case 13:
            return oj(a, b, c);
          case 4:
            return yh(b, b.stateNode.containerInfo), d = b.pendingProps, null === a ? b.child = Ug(b, null, d, c) : Xi(a, b, d, c), b.child;
          case 11:
            return d = b.type, e = b.pendingProps, e = b.elementType === d ? e : Ci(d, e), Yi(a, b, d, e, c);
          case 7:
            return Xi(a, b, b.pendingProps, c), b.child;
          case 8:
            return Xi(a, b, b.pendingProps.children, c), b.child;
          case 12:
            return Xi(a, b, b.pendingProps.children, c), b.child;
          case 10:
            a: {
              d = b.type._context;
              e = b.pendingProps;
              f = b.memoizedProps;
              g = e.value;
              G(Wg, d._currentValue);
              d._currentValue = g;
              if (null !== f) if (He(f.value, g)) {
                if (f.children === e.children && !Wf.current) {
                  b = Zi(a, b, c);
                  break a;
                }
              } else for (f = b.child, null !== f && (f.return = b); null !== f; ) {
                var h = f.dependencies;
                if (null !== h) {
                  g = f.child;
                  for (var k = h.firstContext; null !== k; ) {
                    if (k.context === d) {
                      if (1 === f.tag) {
                        k = mh(-1, c & -c);
                        k.tag = 2;
                        var l = f.updateQueue;
                        if (null !== l) {
                          l = l.shared;
                          var m = l.pending;
                          null === m ? k.next = k : (k.next = m.next, m.next = k);
                          l.pending = k;
                        }
                      }
                      f.lanes |= c;
                      k = f.alternate;
                      null !== k && (k.lanes |= c);
                      bh(
                        f.return,
                        c,
                        b
                      );
                      h.lanes |= c;
                      break;
                    }
                    k = k.next;
                  }
                } else if (10 === f.tag) g = f.type === b.type ? null : f.child;
                else if (18 === f.tag) {
                  g = f.return;
                  if (null === g) throw Error(p(341));
                  g.lanes |= c;
                  h = g.alternate;
                  null !== h && (h.lanes |= c);
                  bh(g, c, b);
                  g = f.sibling;
                } else g = f.child;
                if (null !== g) g.return = f;
                else for (g = f; null !== g; ) {
                  if (g === b) {
                    g = null;
                    break;
                  }
                  f = g.sibling;
                  if (null !== f) {
                    f.return = g.return;
                    g = f;
                    break;
                  }
                  g = g.return;
                }
                f = g;
              }
              Xi(a, b, e.children, c);
              b = b.child;
            }
            return b;
          case 9:
            return e = b.type, d = b.pendingProps.children, ch(b, c), e = eh(e), d = d(e), b.flags |= 1, Xi(a, b, d, c), b.child;
          case 14:
            return d = b.type, e = Ci(d, b.pendingProps), e = Ci(d.type, e), $i(a, b, d, e, c);
          case 15:
            return bj(a, b, b.type, b.pendingProps, c);
          case 17:
            return d = b.type, e = b.pendingProps, e = b.elementType === d ? e : Ci(d, e), ij(a, b), b.tag = 1, Zf(d) ? (a = true, cg(b)) : a = false, ch(b, c), Gi(b, d, e), Ii(b, d, e, c), jj(null, b, d, true, a, c);
          case 19:
            return xj(a, b, c);
          case 22:
            return dj(a, b, c);
        }
        throw Error(p(156, b.tag));
      };
      function Fk(a, b) {
        return ac(a, b);
      }
      function $k(a, b, c, d) {
        this.tag = a;
        this.key = c;
        this.sibling = this.child = this.return = this.stateNode = this.type = this.elementType = null;
        this.index = 0;
        this.ref = null;
        this.pendingProps = b;
        this.dependencies = this.memoizedState = this.updateQueue = this.memoizedProps = null;
        this.mode = d;
        this.subtreeFlags = this.flags = 0;
        this.deletions = null;
        this.childLanes = this.lanes = 0;
        this.alternate = null;
      }
      function Bg(a, b, c, d) {
        return new $k(a, b, c, d);
      }
      function aj(a) {
        a = a.prototype;
        return !(!a || !a.isReactComponent);
      }
      function Zk(a) {
        if ("function" === typeof a) return aj(a) ? 1 : 0;
        if (void 0 !== a && null !== a) {
          a = a.$$typeof;
          if (a === Da) return 11;
          if (a === Ga) return 14;
        }
        return 2;
      }
      function Pg(a, b) {
        var c = a.alternate;
        null === c ? (c = Bg(a.tag, b, a.key, a.mode), c.elementType = a.elementType, c.type = a.type, c.stateNode = a.stateNode, c.alternate = a, a.alternate = c) : (c.pendingProps = b, c.type = a.type, c.flags = 0, c.subtreeFlags = 0, c.deletions = null);
        c.flags = a.flags & 14680064;
        c.childLanes = a.childLanes;
        c.lanes = a.lanes;
        c.child = a.child;
        c.memoizedProps = a.memoizedProps;
        c.memoizedState = a.memoizedState;
        c.updateQueue = a.updateQueue;
        b = a.dependencies;
        c.dependencies = null === b ? null : { lanes: b.lanes, firstContext: b.firstContext };
        c.sibling = a.sibling;
        c.index = a.index;
        c.ref = a.ref;
        return c;
      }
      function Rg(a, b, c, d, e, f) {
        var g = 2;
        d = a;
        if ("function" === typeof a) aj(a) && (g = 1);
        else if ("string" === typeof a) g = 5;
        else a: switch (a) {
          case ya:
            return Tg(c.children, e, f, b);
          case za:
            g = 8;
            e |= 8;
            break;
          case Aa:
            return a = Bg(12, c, b, e | 2), a.elementType = Aa, a.lanes = f, a;
          case Ea:
            return a = Bg(13, c, b, e), a.elementType = Ea, a.lanes = f, a;
          case Fa:
            return a = Bg(19, c, b, e), a.elementType = Fa, a.lanes = f, a;
          case Ia:
            return pj(c, e, f, b);
          default:
            if ("object" === typeof a && null !== a) switch (a.$$typeof) {
              case Ba:
                g = 10;
                break a;
              case Ca:
                g = 9;
                break a;
              case Da:
                g = 11;
                break a;
              case Ga:
                g = 14;
                break a;
              case Ha:
                g = 16;
                d = null;
                break a;
            }
            throw Error(p(130, null == a ? a : typeof a, ""));
        }
        b = Bg(g, c, b, e);
        b.elementType = a;
        b.type = d;
        b.lanes = f;
        return b;
      }
      function Tg(a, b, c, d) {
        a = Bg(7, a, d, b);
        a.lanes = c;
        return a;
      }
      function pj(a, b, c, d) {
        a = Bg(22, a, d, b);
        a.elementType = Ia;
        a.lanes = c;
        a.stateNode = { isHidden: false };
        return a;
      }
      function Qg(a, b, c) {
        a = Bg(6, a, null, b);
        a.lanes = c;
        return a;
      }
      function Sg(a, b, c) {
        b = Bg(4, null !== a.children ? a.children : [], a.key, b);
        b.lanes = c;
        b.stateNode = { containerInfo: a.containerInfo, pendingChildren: null, implementation: a.implementation };
        return b;
      }
      function al(a, b, c, d, e) {
        this.tag = b;
        this.containerInfo = a;
        this.finishedWork = this.pingCache = this.current = this.pendingChildren = null;
        this.timeoutHandle = -1;
        this.callbackNode = this.pendingContext = this.context = null;
        this.callbackPriority = 0;
        this.eventTimes = zc(0);
        this.expirationTimes = zc(-1);
        this.entangledLanes = this.finishedLanes = this.mutableReadLanes = this.expiredLanes = this.pingedLanes = this.suspendedLanes = this.pendingLanes = 0;
        this.entanglements = zc(0);
        this.identifierPrefix = d;
        this.onRecoverableError = e;
        this.mutableSourceEagerHydrationData = null;
      }
      function bl(a, b, c, d, e, f, g, h, k) {
        a = new al(a, b, c, h, k);
        1 === b ? (b = 1, true === f && (b |= 8)) : b = 0;
        f = Bg(3, null, null, b);
        a.current = f;
        f.stateNode = a;
        f.memoizedState = { element: d, isDehydrated: c, cache: null, transitions: null, pendingSuspenseBoundaries: null };
        kh(f);
        return a;
      }
      function cl(a, b, c) {
        var d = 3 < arguments.length && void 0 !== arguments[3] ? arguments[3] : null;
        return { $$typeof: wa, key: null == d ? null : "" + d, children: a, containerInfo: b, implementation: c };
      }
      function dl(a) {
        if (!a) return Vf;
        a = a._reactInternals;
        a: {
          if (Vb(a) !== a || 1 !== a.tag) throw Error(p(170));
          var b = a;
          do {
            switch (b.tag) {
              case 3:
                b = b.stateNode.context;
                break a;
              case 1:
                if (Zf(b.type)) {
                  b = b.stateNode.__reactInternalMemoizedMergedChildContext;
                  break a;
                }
            }
            b = b.return;
          } while (null !== b);
          throw Error(p(171));
        }
        if (1 === a.tag) {
          var c = a.type;
          if (Zf(c)) return bg(a, c, b);
        }
        return b;
      }
      function el(a, b, c, d, e, f, g, h, k) {
        a = bl(c, d, true, a, e, f, g, h, k);
        a.context = dl(null);
        c = a.current;
        d = R();
        e = yi(c);
        f = mh(d, e);
        f.callback = void 0 !== b && null !== b ? b : null;
        nh(c, f, e);
        a.current.lanes = e;
        Ac(a, e, d);
        Dk(a, d);
        return a;
      }
      function fl(a, b, c, d) {
        var e = b.current, f = R(), g = yi(e);
        c = dl(c);
        null === b.context ? b.context = c : b.pendingContext = c;
        b = mh(f, g);
        b.payload = { element: a };
        d = void 0 === d ? null : d;
        null !== d && (b.callback = d);
        a = nh(e, b, g);
        null !== a && (gi(a, e, g, f), oh(a, e, g));
        return g;
      }
      function gl(a) {
        a = a.current;
        if (!a.child) return null;
        switch (a.child.tag) {
          case 5:
            return a.child.stateNode;
          default:
            return a.child.stateNode;
        }
      }
      function hl(a, b) {
        a = a.memoizedState;
        if (null !== a && null !== a.dehydrated) {
          var c = a.retryLane;
          a.retryLane = 0 !== c && c < b ? c : b;
        }
      }
      function il(a, b) {
        hl(a, b);
        (a = a.alternate) && hl(a, b);
      }
      function jl() {
        return null;
      }
      var kl = "function" === typeof reportError ? reportError : function(a) {
        console.error(a);
      };
      function ll(a) {
        this._internalRoot = a;
      }
      ml.prototype.render = ll.prototype.render = function(a) {
        var b = this._internalRoot;
        if (null === b) throw Error(p(409));
        fl(a, b, null, null);
      };
      ml.prototype.unmount = ll.prototype.unmount = function() {
        var a = this._internalRoot;
        if (null !== a) {
          this._internalRoot = null;
          var b = a.containerInfo;
          Rk(function() {
            fl(null, a, null, null);
          });
          b[uf] = null;
        }
      };
      function ml(a) {
        this._internalRoot = a;
      }
      ml.prototype.unstable_scheduleHydration = function(a) {
        if (a) {
          var b = Hc();
          a = { blockedOn: null, target: a, priority: b };
          for (var c = 0; c < Qc.length && 0 !== b && b < Qc[c].priority; c++) ;
          Qc.splice(c, 0, a);
          0 === c && Vc(a);
        }
      };
      function nl(a) {
        return !(!a || 1 !== a.nodeType && 9 !== a.nodeType && 11 !== a.nodeType);
      }
      function ol(a) {
        return !(!a || 1 !== a.nodeType && 9 !== a.nodeType && 11 !== a.nodeType && (8 !== a.nodeType || " react-mount-point-unstable " !== a.nodeValue));
      }
      function pl() {
      }
      function ql(a, b, c, d, e) {
        if (e) {
          if ("function" === typeof d) {
            var f = d;
            d = function() {
              var a2 = gl(g);
              f.call(a2);
            };
          }
          var g = el(b, d, a, 0, null, false, false, "", pl);
          a._reactRootContainer = g;
          a[uf] = g.current;
          sf(8 === a.nodeType ? a.parentNode : a);
          Rk();
          return g;
        }
        for (; e = a.lastChild; ) a.removeChild(e);
        if ("function" === typeof d) {
          var h = d;
          d = function() {
            var a2 = gl(k);
            h.call(a2);
          };
        }
        var k = bl(a, 0, false, null, null, false, false, "", pl);
        a._reactRootContainer = k;
        a[uf] = k.current;
        sf(8 === a.nodeType ? a.parentNode : a);
        Rk(function() {
          fl(b, k, c, d);
        });
        return k;
      }
      function rl(a, b, c, d, e) {
        var f = c._reactRootContainer;
        if (f) {
          var g = f;
          if ("function" === typeof e) {
            var h = e;
            e = function() {
              var a2 = gl(g);
              h.call(a2);
            };
          }
          fl(b, g, a, e);
        } else g = ql(c, b, a, e, d);
        return gl(g);
      }
      Ec = function(a) {
        switch (a.tag) {
          case 3:
            var b = a.stateNode;
            if (b.current.memoizedState.isDehydrated) {
              var c = tc(b.pendingLanes);
              0 !== c && (Cc(b, c | 1), Dk(b, B()), 0 === (K & 6) && (Gj = B() + 500, jg()));
            }
            break;
          case 13:
            Rk(function() {
              var b2 = ih(a, 1);
              if (null !== b2) {
                var c2 = R();
                gi(b2, a, 1, c2);
              }
            }), il(a, 1);
        }
      };
      Fc = function(a) {
        if (13 === a.tag) {
          var b = ih(a, 134217728);
          if (null !== b) {
            var c = R();
            gi(b, a, 134217728, c);
          }
          il(a, 134217728);
        }
      };
      Gc = function(a) {
        if (13 === a.tag) {
          var b = yi(a), c = ih(a, b);
          if (null !== c) {
            var d = R();
            gi(c, a, b, d);
          }
          il(a, b);
        }
      };
      Hc = function() {
        return C;
      };
      Ic = function(a, b) {
        var c = C;
        try {
          return C = a, b();
        } finally {
          C = c;
        }
      };
      yb = function(a, b, c) {
        switch (b) {
          case "input":
            bb(a, c);
            b = c.name;
            if ("radio" === c.type && null != b) {
              for (c = a; c.parentNode; ) c = c.parentNode;
              c = c.querySelectorAll("input[name=" + JSON.stringify("" + b) + '][type="radio"]');
              for (b = 0; b < c.length; b++) {
                var d = c[b];
                if (d !== a && d.form === a.form) {
                  var e = Db(d);
                  if (!e) throw Error(p(90));
                  Wa(d);
                  bb(d, e);
                }
              }
            }
            break;
          case "textarea":
            ib(a, c);
            break;
          case "select":
            b = c.value, null != b && fb(a, !!c.multiple, b, false);
        }
      };
      Gb = Qk;
      Hb = Rk;
      var sl = { usingClientEntryPoint: false, Events: [Cb, ue, Db, Eb, Fb, Qk] };
      var tl = { findFiberByHostInstance: Wc, bundleType: 0, version: "18.3.1", rendererPackageName: "react-dom" };
      var ul = { bundleType: tl.bundleType, version: tl.version, rendererPackageName: tl.rendererPackageName, rendererConfig: tl.rendererConfig, overrideHookState: null, overrideHookStateDeletePath: null, overrideHookStateRenamePath: null, overrideProps: null, overridePropsDeletePath: null, overridePropsRenamePath: null, setErrorHandler: null, setSuspenseHandler: null, scheduleUpdate: null, currentDispatcherRef: ua.ReactCurrentDispatcher, findHostInstanceByFiber: function(a) {
        a = Zb(a);
        return null === a ? null : a.stateNode;
      }, findFiberByHostInstance: tl.findFiberByHostInstance || jl, findHostInstancesForRefresh: null, scheduleRefresh: null, scheduleRoot: null, setRefreshHandler: null, getCurrentFiber: null, reconcilerVersion: "18.3.1-next-f1338f8080-20240426" };
      if ("undefined" !== typeof __REACT_DEVTOOLS_GLOBAL_HOOK__) {
        vl = __REACT_DEVTOOLS_GLOBAL_HOOK__;
        if (!vl.isDisabled && vl.supportsFiber) try {
          kc = vl.inject(ul), lc = vl;
        } catch (a) {
        }
      }
      var vl;
      exports.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED = sl;
      exports.createPortal = function(a, b) {
        var c = 2 < arguments.length && void 0 !== arguments[2] ? arguments[2] : null;
        if (!nl(b)) throw Error(p(200));
        return cl(a, b, null, c);
      };
      exports.createRoot = function(a, b) {
        if (!nl(a)) throw Error(p(299));
        var c = false, d = "", e = kl;
        null !== b && void 0 !== b && (true === b.unstable_strictMode && (c = true), void 0 !== b.identifierPrefix && (d = b.identifierPrefix), void 0 !== b.onRecoverableError && (e = b.onRecoverableError));
        b = bl(a, 1, false, null, null, c, false, d, e);
        a[uf] = b.current;
        sf(8 === a.nodeType ? a.parentNode : a);
        return new ll(b);
      };
      exports.findDOMNode = function(a) {
        if (null == a) return null;
        if (1 === a.nodeType) return a;
        var b = a._reactInternals;
        if (void 0 === b) {
          if ("function" === typeof a.render) throw Error(p(188));
          a = Object.keys(a).join(",");
          throw Error(p(268, a));
        }
        a = Zb(b);
        a = null === a ? null : a.stateNode;
        return a;
      };
      exports.flushSync = function(a) {
        return Rk(a);
      };
      exports.hydrate = function(a, b, c) {
        if (!ol(b)) throw Error(p(200));
        return rl(null, a, b, true, c);
      };
      exports.hydrateRoot = function(a, b, c) {
        if (!nl(a)) throw Error(p(405));
        var d = null != c && c.hydratedSources || null, e = false, f = "", g = kl;
        null !== c && void 0 !== c && (true === c.unstable_strictMode && (e = true), void 0 !== c.identifierPrefix && (f = c.identifierPrefix), void 0 !== c.onRecoverableError && (g = c.onRecoverableError));
        b = el(b, null, a, 1, null != c ? c : null, e, false, f, g);
        a[uf] = b.current;
        sf(a);
        if (d) for (a = 0; a < d.length; a++) c = d[a], e = c._getVersion, e = e(c._source), null == b.mutableSourceEagerHydrationData ? b.mutableSourceEagerHydrationData = [c, e] : b.mutableSourceEagerHydrationData.push(
          c,
          e
        );
        return new ml(b);
      };
      exports.render = function(a, b, c) {
        if (!ol(b)) throw Error(p(200));
        return rl(null, a, b, false, c);
      };
      exports.unmountComponentAtNode = function(a) {
        if (!ol(a)) throw Error(p(40));
        return a._reactRootContainer ? (Rk(function() {
          rl(null, null, a, false, function() {
            a._reactRootContainer = null;
            a[uf] = null;
          });
        }), true) : false;
      };
      exports.unstable_batchedUpdates = Qk;
      exports.unstable_renderSubtreeIntoContainer = function(a, b, c, d) {
        if (!ol(c)) throw Error(p(200));
        if (null == a || void 0 === a._reactInternals) throw Error(p(38));
        return rl(a, b, c, false, d);
      };
      exports.version = "18.3.1-next-f1338f8080-20240426";
    }
  });

  // node_modules/react-dom/index.js
  var require_react_dom = __commonJS({
    "node_modules/react-dom/index.js"(exports, module) {
      "use strict";
      function checkDCE() {
        if (typeof __REACT_DEVTOOLS_GLOBAL_HOOK__ === "undefined" || typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE !== "function") {
          return;
        }
        if (false) {
          throw new Error("^_^");
        }
        try {
          __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(checkDCE);
        } catch (err) {
          console.error(err);
        }
      }
      if (true) {
        checkDCE();
        module.exports = require_react_dom_production_min();
      } else {
        module.exports = null;
      }
    }
  });

  // node_modules/react-dom/client.js
  var require_client = __commonJS({
    "node_modules/react-dom/client.js"(exports) {
      "use strict";
      var m = require_react_dom();
      if (true) {
        exports.createRoot = m.createRoot;
        exports.hydrateRoot = m.hydrateRoot;
      } else {
        i = m.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED;
        exports.createRoot = function(c, o) {
          i.usingClientEntryPoint = true;
          try {
            return m.createRoot(c, o);
          } finally {
            i.usingClientEntryPoint = false;
          }
        };
        exports.hydrateRoot = function(c, h, o) {
          i.usingClientEntryPoint = true;
          try {
            return m.hydrateRoot(c, h, o);
          } finally {
            i.usingClientEntryPoint = false;
          }
        };
      }
      var i;
    }
  });

  // node_modules/react/cjs/react-jsx-runtime.production.min.js
  var require_react_jsx_runtime_production_min = __commonJS({
    "node_modules/react/cjs/react-jsx-runtime.production.min.js"(exports) {
      "use strict";
      var f = require_react();
      var k = Symbol.for("react.element");
      var l = Symbol.for("react.fragment");
      var m = Object.prototype.hasOwnProperty;
      var n = f.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner;
      var p = { key: true, ref: true, __self: true, __source: true };
      function q(c, a, g) {
        var b, d = {}, e = null, h = null;
        void 0 !== g && (e = "" + g);
        void 0 !== a.key && (e = "" + a.key);
        void 0 !== a.ref && (h = a.ref);
        for (b in a) m.call(a, b) && !p.hasOwnProperty(b) && (d[b] = a[b]);
        if (c && c.defaultProps) for (b in a = c.defaultProps, a) void 0 === d[b] && (d[b] = a[b]);
        return { $$typeof: k, type: c, key: e, ref: h, props: d, _owner: n.current };
      }
      exports.Fragment = l;
      exports.jsx = q;
      exports.jsxs = q;
    }
  });

  // node_modules/react/jsx-runtime.js
  var require_jsx_runtime = __commonJS({
    "node_modules/react/jsx-runtime.js"(exports, module) {
      "use strict";
      if (true) {
        module.exports = require_react_jsx_runtime_production_min();
      } else {
        module.exports = null;
      }
    }
  });

  // src/index.jsx
  var import_react9 = __toESM(require_react(), 1);
  var import_client = __toESM(require_client(), 1);

  // src/components/ui/button.jsx
  var import_react = __toESM(require_react(), 1);

  // node_modules/@radix-ui/react-slot/dist/index.mjs
  var React2 = __toESM(require_react(), 1);

  // node_modules/@radix-ui/react-compose-refs/dist/index.mjs
  var React = __toESM(require_react(), 1);
  function setRef(ref, value) {
    if (typeof ref === "function") {
      return ref(value);
    } else if (ref !== null && ref !== void 0) {
      ref.current = value;
    }
  }
  function composeRefs(...refs) {
    return (node) => {
      let hasCleanup = false;
      const cleanups = refs.map((ref) => {
        const cleanup = setRef(ref, node);
        if (!hasCleanup && typeof cleanup == "function") {
          hasCleanup = true;
        }
        return cleanup;
      });
      if (hasCleanup) {
        return () => {
          for (let i = 0; i < cleanups.length; i++) {
            const cleanup = cleanups[i];
            if (typeof cleanup == "function") {
              cleanup();
            } else {
              setRef(refs[i], null);
            }
          }
        };
      }
    };
  }
  function useComposedRefs(...refs) {
    return React.useCallback(composeRefs(...refs), refs);
  }

  // node_modules/@radix-ui/react-slot/dist/index.mjs
  var import_jsx_runtime = __toESM(require_jsx_runtime(), 1);
  var REACT_LAZY_TYPE = Symbol.for("react.lazy");
  var use = React2[" use ".trim().toString()];
  function isPromiseLike(value) {
    return typeof value === "object" && value !== null && "then" in value;
  }
  function isLazyComponent(element) {
    return element != null && typeof element === "object" && "$$typeof" in element && element.$$typeof === REACT_LAZY_TYPE && "_payload" in element && isPromiseLike(element._payload);
  }
  // @__NO_SIDE_EFFECTS__
  function createSlot(ownerName) {
    const SlotClone = /* @__PURE__ */ createSlotClone(ownerName);
    const Slot22 = React2.forwardRef((props, forwardedRef) => {
      let { children, ...slotProps } = props;
      if (isLazyComponent(children) && typeof use === "function") {
        children = use(children._payload);
      }
      const childrenArray = React2.Children.toArray(children);
      const slottable = childrenArray.find(isSlottable);
      if (slottable) {
        const newElement = slottable.props.children;
        const newChildren = childrenArray.map((child) => {
          if (child === slottable) {
            if (React2.Children.count(newElement) > 1) return React2.Children.only(null);
            return React2.isValidElement(newElement) ? newElement.props.children : null;
          } else {
            return child;
          }
        });
        return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(SlotClone, { ...slotProps, ref: forwardedRef, children: React2.isValidElement(newElement) ? React2.cloneElement(newElement, void 0, newChildren) : null });
      }
      return /* @__PURE__ */ (0, import_jsx_runtime.jsx)(SlotClone, { ...slotProps, ref: forwardedRef, children });
    });
    Slot22.displayName = `${ownerName}.Slot`;
    return Slot22;
  }
  var Slot = /* @__PURE__ */ createSlot("Slot");
  // @__NO_SIDE_EFFECTS__
  function createSlotClone(ownerName) {
    const SlotClone = React2.forwardRef((props, forwardedRef) => {
      let { children, ...slotProps } = props;
      if (isLazyComponent(children) && typeof use === "function") {
        children = use(children._payload);
      }
      if (React2.isValidElement(children)) {
        const childrenRef = getElementRef(children);
        const props2 = mergeProps(slotProps, children.props);
        if (children.type !== React2.Fragment) {
          props2.ref = forwardedRef ? composeRefs(forwardedRef, childrenRef) : childrenRef;
        }
        return React2.cloneElement(children, props2);
      }
      return React2.Children.count(children) > 1 ? React2.Children.only(null) : null;
    });
    SlotClone.displayName = `${ownerName}.SlotClone`;
    return SlotClone;
  }
  var SLOTTABLE_IDENTIFIER = Symbol("radix.slottable");
  function isSlottable(child) {
    return React2.isValidElement(child) && typeof child.type === "function" && "__radixId" in child.type && child.type.__radixId === SLOTTABLE_IDENTIFIER;
  }
  function mergeProps(slotProps, childProps) {
    const overrideProps = { ...childProps };
    for (const propName in childProps) {
      const slotPropValue = slotProps[propName];
      const childPropValue = childProps[propName];
      const isHandler = /^on[A-Z]/.test(propName);
      if (isHandler) {
        if (slotPropValue && childPropValue) {
          overrideProps[propName] = (...args) => {
            const result = childPropValue(...args);
            slotPropValue(...args);
            return result;
          };
        } else if (slotPropValue) {
          overrideProps[propName] = slotPropValue;
        }
      } else if (propName === "style") {
        overrideProps[propName] = { ...slotPropValue, ...childPropValue };
      } else if (propName === "className") {
        overrideProps[propName] = [slotPropValue, childPropValue].filter(Boolean).join(" ");
      }
    }
    return { ...slotProps, ...overrideProps };
  }
  function getElementRef(element) {
    let getter = Object.getOwnPropertyDescriptor(element.props, "ref")?.get;
    let mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.ref;
    }
    getter = Object.getOwnPropertyDescriptor(element, "ref")?.get;
    mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.props.ref;
    }
    return element.props.ref || element.ref;
  }

  // node_modules/clsx/dist/clsx.mjs
  function r(e) {
    var t, f, n = "";
    if ("string" == typeof e || "number" == typeof e) n += e;
    else if ("object" == typeof e) if (Array.isArray(e)) {
      var o = e.length;
      for (t = 0; t < o; t++) e[t] && (f = r(e[t])) && (n && (n += " "), n += f);
    } else for (f in e) e[f] && (n && (n += " "), n += f);
    return n;
  }
  function clsx() {
    for (var e, t, f = 0, n = "", o = arguments.length; f < o; f++) (e = arguments[f]) && (t = r(e)) && (n && (n += " "), n += t);
    return n;
  }

  // node_modules/class-variance-authority/dist/index.mjs
  var falsyToString = (value) => typeof value === "boolean" ? `${value}` : value === 0 ? "0" : value;
  var cx = clsx;
  var cva = (base, config) => (props) => {
    var _config_compoundVariants;
    if ((config === null || config === void 0 ? void 0 : config.variants) == null) return cx(base, props === null || props === void 0 ? void 0 : props.class, props === null || props === void 0 ? void 0 : props.className);
    const { variants, defaultVariants } = config;
    const getVariantClassNames = Object.keys(variants).map((variant) => {
      const variantProp = props === null || props === void 0 ? void 0 : props[variant];
      const defaultVariantProp = defaultVariants === null || defaultVariants === void 0 ? void 0 : defaultVariants[variant];
      if (variantProp === null) return null;
      const variantKey = falsyToString(variantProp) || falsyToString(defaultVariantProp);
      return variants[variant][variantKey];
    });
    const propsWithoutUndefined = props && Object.entries(props).reduce((acc, param) => {
      let [key, value] = param;
      if (value === void 0) {
        return acc;
      }
      acc[key] = value;
      return acc;
    }, {});
    const getCompoundVariantClassNames = config === null || config === void 0 ? void 0 : (_config_compoundVariants = config.compoundVariants) === null || _config_compoundVariants === void 0 ? void 0 : _config_compoundVariants.reduce((acc, param) => {
      let { class: cvClass, className: cvClassName, ...compoundVariantOptions } = param;
      return Object.entries(compoundVariantOptions).every((param2) => {
        let [key, value] = param2;
        return Array.isArray(value) ? value.includes({
          ...defaultVariants,
          ...propsWithoutUndefined
        }[key]) : {
          ...defaultVariants,
          ...propsWithoutUndefined
        }[key] === value;
      }) ? [
        ...acc,
        cvClass,
        cvClassName
      ] : acc;
    }, []);
    return cx(base, getVariantClassNames, getCompoundVariantClassNames, props === null || props === void 0 ? void 0 : props.class, props === null || props === void 0 ? void 0 : props.className);
  };

  // node_modules/tailwind-merge/dist/bundle-mjs.mjs
  var concatArrays = (array1, array2) => {
    const combinedArray = new Array(array1.length + array2.length);
    for (let i = 0; i < array1.length; i++) {
      combinedArray[i] = array1[i];
    }
    for (let i = 0; i < array2.length; i++) {
      combinedArray[array1.length + i] = array2[i];
    }
    return combinedArray;
  };
  var createClassValidatorObject = (classGroupId, validator) => ({
    classGroupId,
    validator
  });
  var createClassPartObject = (nextPart = /* @__PURE__ */ new Map(), validators = null, classGroupId) => ({
    nextPart,
    validators,
    classGroupId
  });
  var CLASS_PART_SEPARATOR = "-";
  var EMPTY_CONFLICTS = [];
  var ARBITRARY_PROPERTY_PREFIX = "arbitrary..";
  var createClassGroupUtils = (config) => {
    const classMap = createClassMap(config);
    const {
      conflictingClassGroups,
      conflictingClassGroupModifiers
    } = config;
    const getClassGroupId = (className) => {
      if (className.startsWith("[") && className.endsWith("]")) {
        return getGroupIdForArbitraryProperty(className);
      }
      const classParts = className.split(CLASS_PART_SEPARATOR);
      const startIndex = classParts[0] === "" && classParts.length > 1 ? 1 : 0;
      return getGroupRecursive(classParts, startIndex, classMap);
    };
    const getConflictingClassGroupIds = (classGroupId, hasPostfixModifier) => {
      if (hasPostfixModifier) {
        const modifierConflicts = conflictingClassGroupModifiers[classGroupId];
        const baseConflicts = conflictingClassGroups[classGroupId];
        if (modifierConflicts) {
          if (baseConflicts) {
            return concatArrays(baseConflicts, modifierConflicts);
          }
          return modifierConflicts;
        }
        return baseConflicts || EMPTY_CONFLICTS;
      }
      return conflictingClassGroups[classGroupId] || EMPTY_CONFLICTS;
    };
    return {
      getClassGroupId,
      getConflictingClassGroupIds
    };
  };
  var getGroupRecursive = (classParts, startIndex, classPartObject) => {
    const classPathsLength = classParts.length - startIndex;
    if (classPathsLength === 0) {
      return classPartObject.classGroupId;
    }
    const currentClassPart = classParts[startIndex];
    const nextClassPartObject = classPartObject.nextPart.get(currentClassPart);
    if (nextClassPartObject) {
      const result = getGroupRecursive(classParts, startIndex + 1, nextClassPartObject);
      if (result) return result;
    }
    const validators = classPartObject.validators;
    if (validators === null) {
      return void 0;
    }
    const classRest = startIndex === 0 ? classParts.join(CLASS_PART_SEPARATOR) : classParts.slice(startIndex).join(CLASS_PART_SEPARATOR);
    const validatorsLength = validators.length;
    for (let i = 0; i < validatorsLength; i++) {
      const validatorObj = validators[i];
      if (validatorObj.validator(classRest)) {
        return validatorObj.classGroupId;
      }
    }
    return void 0;
  };
  var getGroupIdForArbitraryProperty = (className) => className.slice(1, -1).indexOf(":") === -1 ? void 0 : (() => {
    const content = className.slice(1, -1);
    const colonIndex = content.indexOf(":");
    const property = content.slice(0, colonIndex);
    return property ? ARBITRARY_PROPERTY_PREFIX + property : void 0;
  })();
  var createClassMap = (config) => {
    const {
      theme,
      classGroups
    } = config;
    return processClassGroups(classGroups, theme);
  };
  var processClassGroups = (classGroups, theme) => {
    const classMap = createClassPartObject();
    for (const classGroupId in classGroups) {
      const group = classGroups[classGroupId];
      processClassesRecursively(group, classMap, classGroupId, theme);
    }
    return classMap;
  };
  var processClassesRecursively = (classGroup, classPartObject, classGroupId, theme) => {
    const len = classGroup.length;
    for (let i = 0; i < len; i++) {
      const classDefinition = classGroup[i];
      processClassDefinition(classDefinition, classPartObject, classGroupId, theme);
    }
  };
  var processClassDefinition = (classDefinition, classPartObject, classGroupId, theme) => {
    if (typeof classDefinition === "string") {
      processStringDefinition(classDefinition, classPartObject, classGroupId);
      return;
    }
    if (typeof classDefinition === "function") {
      processFunctionDefinition(classDefinition, classPartObject, classGroupId, theme);
      return;
    }
    processObjectDefinition(classDefinition, classPartObject, classGroupId, theme);
  };
  var processStringDefinition = (classDefinition, classPartObject, classGroupId) => {
    const classPartObjectToEdit = classDefinition === "" ? classPartObject : getPart(classPartObject, classDefinition);
    classPartObjectToEdit.classGroupId = classGroupId;
  };
  var processFunctionDefinition = (classDefinition, classPartObject, classGroupId, theme) => {
    if (isThemeGetter(classDefinition)) {
      processClassesRecursively(classDefinition(theme), classPartObject, classGroupId, theme);
      return;
    }
    if (classPartObject.validators === null) {
      classPartObject.validators = [];
    }
    classPartObject.validators.push(createClassValidatorObject(classGroupId, classDefinition));
  };
  var processObjectDefinition = (classDefinition, classPartObject, classGroupId, theme) => {
    const entries = Object.entries(classDefinition);
    const len = entries.length;
    for (let i = 0; i < len; i++) {
      const [key, value] = entries[i];
      processClassesRecursively(value, getPart(classPartObject, key), classGroupId, theme);
    }
  };
  var getPart = (classPartObject, path) => {
    let current = classPartObject;
    const parts = path.split(CLASS_PART_SEPARATOR);
    const len = parts.length;
    for (let i = 0; i < len; i++) {
      const part = parts[i];
      let next = current.nextPart.get(part);
      if (!next) {
        next = createClassPartObject();
        current.nextPart.set(part, next);
      }
      current = next;
    }
    return current;
  };
  var isThemeGetter = (func) => "isThemeGetter" in func && func.isThemeGetter === true;
  var createLruCache = (maxCacheSize) => {
    if (maxCacheSize < 1) {
      return {
        get: () => void 0,
        set: () => {
        }
      };
    }
    let cacheSize = 0;
    let cache = /* @__PURE__ */ Object.create(null);
    let previousCache = /* @__PURE__ */ Object.create(null);
    const update = (key, value) => {
      cache[key] = value;
      cacheSize++;
      if (cacheSize > maxCacheSize) {
        cacheSize = 0;
        previousCache = cache;
        cache = /* @__PURE__ */ Object.create(null);
      }
    };
    return {
      get(key) {
        let value = cache[key];
        if (value !== void 0) {
          return value;
        }
        if ((value = previousCache[key]) !== void 0) {
          update(key, value);
          return value;
        }
      },
      set(key, value) {
        if (key in cache) {
          cache[key] = value;
        } else {
          update(key, value);
        }
      }
    };
  };
  var IMPORTANT_MODIFIER = "!";
  var MODIFIER_SEPARATOR = ":";
  var EMPTY_MODIFIERS = [];
  var createResultObject = (modifiers, hasImportantModifier, baseClassName, maybePostfixModifierPosition, isExternal) => ({
    modifiers,
    hasImportantModifier,
    baseClassName,
    maybePostfixModifierPosition,
    isExternal
  });
  var createParseClassName = (config) => {
    const {
      prefix,
      experimentalParseClassName
    } = config;
    let parseClassName = (className) => {
      const modifiers = [];
      let bracketDepth = 0;
      let parenDepth = 0;
      let modifierStart = 0;
      let postfixModifierPosition;
      const len = className.length;
      for (let index = 0; index < len; index++) {
        const currentCharacter = className[index];
        if (bracketDepth === 0 && parenDepth === 0) {
          if (currentCharacter === MODIFIER_SEPARATOR) {
            modifiers.push(className.slice(modifierStart, index));
            modifierStart = index + 1;
            continue;
          }
          if (currentCharacter === "/") {
            postfixModifierPosition = index;
            continue;
          }
        }
        if (currentCharacter === "[") bracketDepth++;
        else if (currentCharacter === "]") bracketDepth--;
        else if (currentCharacter === "(") parenDepth++;
        else if (currentCharacter === ")") parenDepth--;
      }
      const baseClassNameWithImportantModifier = modifiers.length === 0 ? className : className.slice(modifierStart);
      let baseClassName = baseClassNameWithImportantModifier;
      let hasImportantModifier = false;
      if (baseClassNameWithImportantModifier.endsWith(IMPORTANT_MODIFIER)) {
        baseClassName = baseClassNameWithImportantModifier.slice(0, -1);
        hasImportantModifier = true;
      } else if (
        /**
         * In Tailwind CSS v3 the important modifier was at the start of the base class name. This is still supported for legacy reasons.
         * @see https://github.com/dcastil/tailwind-merge/issues/513#issuecomment-2614029864
         */
        baseClassNameWithImportantModifier.startsWith(IMPORTANT_MODIFIER)
      ) {
        baseClassName = baseClassNameWithImportantModifier.slice(1);
        hasImportantModifier = true;
      }
      const maybePostfixModifierPosition = postfixModifierPosition && postfixModifierPosition > modifierStart ? postfixModifierPosition - modifierStart : void 0;
      return createResultObject(modifiers, hasImportantModifier, baseClassName, maybePostfixModifierPosition);
    };
    if (prefix) {
      const fullPrefix = prefix + MODIFIER_SEPARATOR;
      const parseClassNameOriginal = parseClassName;
      parseClassName = (className) => className.startsWith(fullPrefix) ? parseClassNameOriginal(className.slice(fullPrefix.length)) : createResultObject(EMPTY_MODIFIERS, false, className, void 0, true);
    }
    if (experimentalParseClassName) {
      const parseClassNameOriginal = parseClassName;
      parseClassName = (className) => experimentalParseClassName({
        className,
        parseClassName: parseClassNameOriginal
      });
    }
    return parseClassName;
  };
  var createSortModifiers = (config) => {
    const modifierWeights = /* @__PURE__ */ new Map();
    config.orderSensitiveModifiers.forEach((mod, index) => {
      modifierWeights.set(mod, 1e6 + index);
    });
    return (modifiers) => {
      const result = [];
      let currentSegment = [];
      for (let i = 0; i < modifiers.length; i++) {
        const modifier = modifiers[i];
        const isArbitrary = modifier[0] === "[";
        const isOrderSensitive = modifierWeights.has(modifier);
        if (isArbitrary || isOrderSensitive) {
          if (currentSegment.length > 0) {
            currentSegment.sort();
            result.push(...currentSegment);
            currentSegment = [];
          }
          result.push(modifier);
        } else {
          currentSegment.push(modifier);
        }
      }
      if (currentSegment.length > 0) {
        currentSegment.sort();
        result.push(...currentSegment);
      }
      return result;
    };
  };
  var createConfigUtils = (config) => ({
    cache: createLruCache(config.cacheSize),
    parseClassName: createParseClassName(config),
    sortModifiers: createSortModifiers(config),
    ...createClassGroupUtils(config)
  });
  var SPLIT_CLASSES_REGEX = /\s+/;
  var mergeClassList = (classList, configUtils) => {
    const {
      parseClassName,
      getClassGroupId,
      getConflictingClassGroupIds,
      sortModifiers
    } = configUtils;
    const classGroupsInConflict = [];
    const classNames = classList.trim().split(SPLIT_CLASSES_REGEX);
    let result = "";
    for (let index = classNames.length - 1; index >= 0; index -= 1) {
      const originalClassName = classNames[index];
      const {
        isExternal,
        modifiers,
        hasImportantModifier,
        baseClassName,
        maybePostfixModifierPosition
      } = parseClassName(originalClassName);
      if (isExternal) {
        result = originalClassName + (result.length > 0 ? " " + result : result);
        continue;
      }
      let hasPostfixModifier = !!maybePostfixModifierPosition;
      let classGroupId = getClassGroupId(hasPostfixModifier ? baseClassName.substring(0, maybePostfixModifierPosition) : baseClassName);
      if (!classGroupId) {
        if (!hasPostfixModifier) {
          result = originalClassName + (result.length > 0 ? " " + result : result);
          continue;
        }
        classGroupId = getClassGroupId(baseClassName);
        if (!classGroupId) {
          result = originalClassName + (result.length > 0 ? " " + result : result);
          continue;
        }
        hasPostfixModifier = false;
      }
      const variantModifier = modifiers.length === 0 ? "" : modifiers.length === 1 ? modifiers[0] : sortModifiers(modifiers).join(":");
      const modifierId = hasImportantModifier ? variantModifier + IMPORTANT_MODIFIER : variantModifier;
      const classId = modifierId + classGroupId;
      if (classGroupsInConflict.indexOf(classId) > -1) {
        continue;
      }
      classGroupsInConflict.push(classId);
      const conflictGroups = getConflictingClassGroupIds(classGroupId, hasPostfixModifier);
      for (let i = 0; i < conflictGroups.length; ++i) {
        const group = conflictGroups[i];
        classGroupsInConflict.push(modifierId + group);
      }
      result = originalClassName + (result.length > 0 ? " " + result : result);
    }
    return result;
  };
  var twJoin = (...classLists) => {
    let index = 0;
    let argument;
    let resolvedValue;
    let string = "";
    while (index < classLists.length) {
      if (argument = classLists[index++]) {
        if (resolvedValue = toValue(argument)) {
          string && (string += " ");
          string += resolvedValue;
        }
      }
    }
    return string;
  };
  var toValue = (mix) => {
    if (typeof mix === "string") {
      return mix;
    }
    let resolvedValue;
    let string = "";
    for (let k = 0; k < mix.length; k++) {
      if (mix[k]) {
        if (resolvedValue = toValue(mix[k])) {
          string && (string += " ");
          string += resolvedValue;
        }
      }
    }
    return string;
  };
  var createTailwindMerge = (createConfigFirst, ...createConfigRest) => {
    let configUtils;
    let cacheGet;
    let cacheSet;
    let functionToCall;
    const initTailwindMerge = (classList) => {
      const config = createConfigRest.reduce((previousConfig, createConfigCurrent) => createConfigCurrent(previousConfig), createConfigFirst());
      configUtils = createConfigUtils(config);
      cacheGet = configUtils.cache.get;
      cacheSet = configUtils.cache.set;
      functionToCall = tailwindMerge;
      return tailwindMerge(classList);
    };
    const tailwindMerge = (classList) => {
      const cachedResult = cacheGet(classList);
      if (cachedResult) {
        return cachedResult;
      }
      const result = mergeClassList(classList, configUtils);
      cacheSet(classList, result);
      return result;
    };
    functionToCall = initTailwindMerge;
    return (...args) => functionToCall(twJoin(...args));
  };
  var fallbackThemeArr = [];
  var fromTheme = (key) => {
    const themeGetter = (theme) => theme[key] || fallbackThemeArr;
    themeGetter.isThemeGetter = true;
    return themeGetter;
  };
  var arbitraryValueRegex = /^\[(?:(\w[\w-]*):)?(.+)\]$/i;
  var arbitraryVariableRegex = /^\((?:(\w[\w-]*):)?(.+)\)$/i;
  var fractionRegex = /^\d+(?:\.\d+)?\/\d+(?:\.\d+)?$/;
  var tshirtUnitRegex = /^(\d+(\.\d+)?)?(xs|sm|md|lg|xl)$/;
  var lengthUnitRegex = /\d+(%|px|r?em|[sdl]?v([hwib]|min|max)|pt|pc|in|cm|mm|cap|ch|ex|r?lh|cq(w|h|i|b|min|max))|\b(calc|min|max|clamp)\(.+\)|^0$/;
  var colorFunctionRegex = /^(rgba?|hsla?|hwb|(ok)?(lab|lch)|color-mix)\(.+\)$/;
  var shadowRegex = /^(inset_)?-?((\d+)?\.?(\d+)[a-z]+|0)_-?((\d+)?\.?(\d+)[a-z]+|0)/;
  var imageRegex = /^(url|image|image-set|cross-fade|element|(repeating-)?(linear|radial|conic)-gradient)\(.+\)$/;
  var isFraction = (value) => fractionRegex.test(value);
  var isNumber = (value) => !!value && !Number.isNaN(Number(value));
  var isInteger = (value) => !!value && Number.isInteger(Number(value));
  var isPercent = (value) => value.endsWith("%") && isNumber(value.slice(0, -1));
  var isTshirtSize = (value) => tshirtUnitRegex.test(value);
  var isAny = () => true;
  var isLengthOnly = (value) => (
    // `colorFunctionRegex` check is necessary because color functions can have percentages in them which which would be incorrectly classified as lengths.
    // For example, `hsl(0 0% 0%)` would be classified as a length without this check.
    // I could also use lookbehind assertion in `lengthUnitRegex` but that isn't supported widely enough.
    lengthUnitRegex.test(value) && !colorFunctionRegex.test(value)
  );
  var isNever = () => false;
  var isShadow = (value) => shadowRegex.test(value);
  var isImage = (value) => imageRegex.test(value);
  var isAnyNonArbitrary = (value) => !isArbitraryValue(value) && !isArbitraryVariable(value);
  var isArbitrarySize = (value) => getIsArbitraryValue(value, isLabelSize, isNever);
  var isArbitraryValue = (value) => arbitraryValueRegex.test(value);
  var isArbitraryLength = (value) => getIsArbitraryValue(value, isLabelLength, isLengthOnly);
  var isArbitraryNumber = (value) => getIsArbitraryValue(value, isLabelNumber, isNumber);
  var isArbitraryWeight = (value) => getIsArbitraryValue(value, isLabelWeight, isAny);
  var isArbitraryFamilyName = (value) => getIsArbitraryValue(value, isLabelFamilyName, isNever);
  var isArbitraryPosition = (value) => getIsArbitraryValue(value, isLabelPosition, isNever);
  var isArbitraryImage = (value) => getIsArbitraryValue(value, isLabelImage, isImage);
  var isArbitraryShadow = (value) => getIsArbitraryValue(value, isLabelShadow, isShadow);
  var isArbitraryVariable = (value) => arbitraryVariableRegex.test(value);
  var isArbitraryVariableLength = (value) => getIsArbitraryVariable(value, isLabelLength);
  var isArbitraryVariableFamilyName = (value) => getIsArbitraryVariable(value, isLabelFamilyName);
  var isArbitraryVariablePosition = (value) => getIsArbitraryVariable(value, isLabelPosition);
  var isArbitraryVariableSize = (value) => getIsArbitraryVariable(value, isLabelSize);
  var isArbitraryVariableImage = (value) => getIsArbitraryVariable(value, isLabelImage);
  var isArbitraryVariableShadow = (value) => getIsArbitraryVariable(value, isLabelShadow, true);
  var isArbitraryVariableWeight = (value) => getIsArbitraryVariable(value, isLabelWeight, true);
  var getIsArbitraryValue = (value, testLabel, testValue) => {
    const result = arbitraryValueRegex.exec(value);
    if (result) {
      if (result[1]) {
        return testLabel(result[1]);
      }
      return testValue(result[2]);
    }
    return false;
  };
  var getIsArbitraryVariable = (value, testLabel, shouldMatchNoLabel = false) => {
    const result = arbitraryVariableRegex.exec(value);
    if (result) {
      if (result[1]) {
        return testLabel(result[1]);
      }
      return shouldMatchNoLabel;
    }
    return false;
  };
  var isLabelPosition = (label) => label === "position" || label === "percentage";
  var isLabelImage = (label) => label === "image" || label === "url";
  var isLabelSize = (label) => label === "length" || label === "size" || label === "bg-size";
  var isLabelLength = (label) => label === "length";
  var isLabelNumber = (label) => label === "number";
  var isLabelFamilyName = (label) => label === "family-name";
  var isLabelWeight = (label) => label === "number" || label === "weight";
  var isLabelShadow = (label) => label === "shadow";
  var getDefaultConfig = () => {
    const themeColor = fromTheme("color");
    const themeFont = fromTheme("font");
    const themeText = fromTheme("text");
    const themeFontWeight = fromTheme("font-weight");
    const themeTracking = fromTheme("tracking");
    const themeLeading = fromTheme("leading");
    const themeBreakpoint = fromTheme("breakpoint");
    const themeContainer = fromTheme("container");
    const themeSpacing = fromTheme("spacing");
    const themeRadius = fromTheme("radius");
    const themeShadow = fromTheme("shadow");
    const themeInsetShadow = fromTheme("inset-shadow");
    const themeTextShadow = fromTheme("text-shadow");
    const themeDropShadow = fromTheme("drop-shadow");
    const themeBlur = fromTheme("blur");
    const themePerspective = fromTheme("perspective");
    const themeAspect = fromTheme("aspect");
    const themeEase = fromTheme("ease");
    const themeAnimate = fromTheme("animate");
    const scaleBreak = () => ["auto", "avoid", "all", "avoid-page", "page", "left", "right", "column"];
    const scalePosition = () => [
      "center",
      "top",
      "bottom",
      "left",
      "right",
      "top-left",
      // Deprecated since Tailwind CSS v4.1.0, see https://github.com/tailwindlabs/tailwindcss/pull/17378
      "left-top",
      "top-right",
      // Deprecated since Tailwind CSS v4.1.0, see https://github.com/tailwindlabs/tailwindcss/pull/17378
      "right-top",
      "bottom-right",
      // Deprecated since Tailwind CSS v4.1.0, see https://github.com/tailwindlabs/tailwindcss/pull/17378
      "right-bottom",
      "bottom-left",
      // Deprecated since Tailwind CSS v4.1.0, see https://github.com/tailwindlabs/tailwindcss/pull/17378
      "left-bottom"
    ];
    const scalePositionWithArbitrary = () => [...scalePosition(), isArbitraryVariable, isArbitraryValue];
    const scaleOverflow = () => ["auto", "hidden", "clip", "visible", "scroll"];
    const scaleOverscroll = () => ["auto", "contain", "none"];
    const scaleUnambiguousSpacing = () => [isArbitraryVariable, isArbitraryValue, themeSpacing];
    const scaleInset = () => [isFraction, "full", "auto", ...scaleUnambiguousSpacing()];
    const scaleGridTemplateColsRows = () => [isInteger, "none", "subgrid", isArbitraryVariable, isArbitraryValue];
    const scaleGridColRowStartAndEnd = () => ["auto", {
      span: ["full", isInteger, isArbitraryVariable, isArbitraryValue]
    }, isInteger, isArbitraryVariable, isArbitraryValue];
    const scaleGridColRowStartOrEnd = () => [isInteger, "auto", isArbitraryVariable, isArbitraryValue];
    const scaleGridAutoColsRows = () => ["auto", "min", "max", "fr", isArbitraryVariable, isArbitraryValue];
    const scaleAlignPrimaryAxis = () => ["start", "end", "center", "between", "around", "evenly", "stretch", "baseline", "center-safe", "end-safe"];
    const scaleAlignSecondaryAxis = () => ["start", "end", "center", "stretch", "center-safe", "end-safe"];
    const scaleMargin = () => ["auto", ...scaleUnambiguousSpacing()];
    const scaleSizing = () => [isFraction, "auto", "full", "dvw", "dvh", "lvw", "lvh", "svw", "svh", "min", "max", "fit", ...scaleUnambiguousSpacing()];
    const scaleSizingInline = () => [isFraction, "screen", "full", "dvw", "lvw", "svw", "min", "max", "fit", ...scaleUnambiguousSpacing()];
    const scaleSizingBlock = () => [isFraction, "screen", "full", "lh", "dvh", "lvh", "svh", "min", "max", "fit", ...scaleUnambiguousSpacing()];
    const scaleColor = () => [themeColor, isArbitraryVariable, isArbitraryValue];
    const scaleBgPosition = () => [...scalePosition(), isArbitraryVariablePosition, isArbitraryPosition, {
      position: [isArbitraryVariable, isArbitraryValue]
    }];
    const scaleBgRepeat = () => ["no-repeat", {
      repeat: ["", "x", "y", "space", "round"]
    }];
    const scaleBgSize = () => ["auto", "cover", "contain", isArbitraryVariableSize, isArbitrarySize, {
      size: [isArbitraryVariable, isArbitraryValue]
    }];
    const scaleGradientStopPosition = () => [isPercent, isArbitraryVariableLength, isArbitraryLength];
    const scaleRadius = () => [
      // Deprecated since Tailwind CSS v4.0.0
      "",
      "none",
      "full",
      themeRadius,
      isArbitraryVariable,
      isArbitraryValue
    ];
    const scaleBorderWidth = () => ["", isNumber, isArbitraryVariableLength, isArbitraryLength];
    const scaleLineStyle = () => ["solid", "dashed", "dotted", "double"];
    const scaleBlendMode = () => ["normal", "multiply", "screen", "overlay", "darken", "lighten", "color-dodge", "color-burn", "hard-light", "soft-light", "difference", "exclusion", "hue", "saturation", "color", "luminosity"];
    const scaleMaskImagePosition = () => [isNumber, isPercent, isArbitraryVariablePosition, isArbitraryPosition];
    const scaleBlur = () => [
      // Deprecated since Tailwind CSS v4.0.0
      "",
      "none",
      themeBlur,
      isArbitraryVariable,
      isArbitraryValue
    ];
    const scaleRotate = () => ["none", isNumber, isArbitraryVariable, isArbitraryValue];
    const scaleScale = () => ["none", isNumber, isArbitraryVariable, isArbitraryValue];
    const scaleSkew = () => [isNumber, isArbitraryVariable, isArbitraryValue];
    const scaleTranslate = () => [isFraction, "full", ...scaleUnambiguousSpacing()];
    return {
      cacheSize: 500,
      theme: {
        animate: ["spin", "ping", "pulse", "bounce"],
        aspect: ["video"],
        blur: [isTshirtSize],
        breakpoint: [isTshirtSize],
        color: [isAny],
        container: [isTshirtSize],
        "drop-shadow": [isTshirtSize],
        ease: ["in", "out", "in-out"],
        font: [isAnyNonArbitrary],
        "font-weight": ["thin", "extralight", "light", "normal", "medium", "semibold", "bold", "extrabold", "black"],
        "inset-shadow": [isTshirtSize],
        leading: ["none", "tight", "snug", "normal", "relaxed", "loose"],
        perspective: ["dramatic", "near", "normal", "midrange", "distant", "none"],
        radius: [isTshirtSize],
        shadow: [isTshirtSize],
        spacing: ["px", isNumber],
        text: [isTshirtSize],
        "text-shadow": [isTshirtSize],
        tracking: ["tighter", "tight", "normal", "wide", "wider", "widest"]
      },
      classGroups: {
        // --------------
        // --- Layout ---
        // --------------
        /**
         * Aspect Ratio
         * @see https://tailwindcss.com/docs/aspect-ratio
         */
        aspect: [{
          aspect: ["auto", "square", isFraction, isArbitraryValue, isArbitraryVariable, themeAspect]
        }],
        /**
         * Container
         * @see https://tailwindcss.com/docs/container
         * @deprecated since Tailwind CSS v4.0.0
         */
        container: ["container"],
        /**
         * Columns
         * @see https://tailwindcss.com/docs/columns
         */
        columns: [{
          columns: [isNumber, isArbitraryValue, isArbitraryVariable, themeContainer]
        }],
        /**
         * Break After
         * @see https://tailwindcss.com/docs/break-after
         */
        "break-after": [{
          "break-after": scaleBreak()
        }],
        /**
         * Break Before
         * @see https://tailwindcss.com/docs/break-before
         */
        "break-before": [{
          "break-before": scaleBreak()
        }],
        /**
         * Break Inside
         * @see https://tailwindcss.com/docs/break-inside
         */
        "break-inside": [{
          "break-inside": ["auto", "avoid", "avoid-page", "avoid-column"]
        }],
        /**
         * Box Decoration Break
         * @see https://tailwindcss.com/docs/box-decoration-break
         */
        "box-decoration": [{
          "box-decoration": ["slice", "clone"]
        }],
        /**
         * Box Sizing
         * @see https://tailwindcss.com/docs/box-sizing
         */
        box: [{
          box: ["border", "content"]
        }],
        /**
         * Display
         * @see https://tailwindcss.com/docs/display
         */
        display: ["block", "inline-block", "inline", "flex", "inline-flex", "table", "inline-table", "table-caption", "table-cell", "table-column", "table-column-group", "table-footer-group", "table-header-group", "table-row-group", "table-row", "flow-root", "grid", "inline-grid", "contents", "list-item", "hidden"],
        /**
         * Screen Reader Only
         * @see https://tailwindcss.com/docs/display#screen-reader-only
         */
        sr: ["sr-only", "not-sr-only"],
        /**
         * Floats
         * @see https://tailwindcss.com/docs/float
         */
        float: [{
          float: ["right", "left", "none", "start", "end"]
        }],
        /**
         * Clear
         * @see https://tailwindcss.com/docs/clear
         */
        clear: [{
          clear: ["left", "right", "both", "none", "start", "end"]
        }],
        /**
         * Isolation
         * @see https://tailwindcss.com/docs/isolation
         */
        isolation: ["isolate", "isolation-auto"],
        /**
         * Object Fit
         * @see https://tailwindcss.com/docs/object-fit
         */
        "object-fit": [{
          object: ["contain", "cover", "fill", "none", "scale-down"]
        }],
        /**
         * Object Position
         * @see https://tailwindcss.com/docs/object-position
         */
        "object-position": [{
          object: scalePositionWithArbitrary()
        }],
        /**
         * Overflow
         * @see https://tailwindcss.com/docs/overflow
         */
        overflow: [{
          overflow: scaleOverflow()
        }],
        /**
         * Overflow X
         * @see https://tailwindcss.com/docs/overflow
         */
        "overflow-x": [{
          "overflow-x": scaleOverflow()
        }],
        /**
         * Overflow Y
         * @see https://tailwindcss.com/docs/overflow
         */
        "overflow-y": [{
          "overflow-y": scaleOverflow()
        }],
        /**
         * Overscroll Behavior
         * @see https://tailwindcss.com/docs/overscroll-behavior
         */
        overscroll: [{
          overscroll: scaleOverscroll()
        }],
        /**
         * Overscroll Behavior X
         * @see https://tailwindcss.com/docs/overscroll-behavior
         */
        "overscroll-x": [{
          "overscroll-x": scaleOverscroll()
        }],
        /**
         * Overscroll Behavior Y
         * @see https://tailwindcss.com/docs/overscroll-behavior
         */
        "overscroll-y": [{
          "overscroll-y": scaleOverscroll()
        }],
        /**
         * Position
         * @see https://tailwindcss.com/docs/position
         */
        position: ["static", "fixed", "absolute", "relative", "sticky"],
        /**
         * Inset
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        inset: [{
          inset: scaleInset()
        }],
        /**
         * Inset Inline
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        "inset-x": [{
          "inset-x": scaleInset()
        }],
        /**
         * Inset Block
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        "inset-y": [{
          "inset-y": scaleInset()
        }],
        /**
         * Inset Inline Start
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         * @todo class group will be renamed to `inset-s` in next major release
         */
        start: [{
          "inset-s": scaleInset(),
          /**
           * @deprecated since Tailwind CSS v4.2.0 in favor of `inset-s-*` utilities.
           * @see https://github.com/tailwindlabs/tailwindcss/pull/19613
           */
          start: scaleInset()
        }],
        /**
         * Inset Inline End
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         * @todo class group will be renamed to `inset-e` in next major release
         */
        end: [{
          "inset-e": scaleInset(),
          /**
           * @deprecated since Tailwind CSS v4.2.0 in favor of `inset-e-*` utilities.
           * @see https://github.com/tailwindlabs/tailwindcss/pull/19613
           */
          end: scaleInset()
        }],
        /**
         * Inset Block Start
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        "inset-bs": [{
          "inset-bs": scaleInset()
        }],
        /**
         * Inset Block End
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        "inset-be": [{
          "inset-be": scaleInset()
        }],
        /**
         * Top
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        top: [{
          top: scaleInset()
        }],
        /**
         * Right
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        right: [{
          right: scaleInset()
        }],
        /**
         * Bottom
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        bottom: [{
          bottom: scaleInset()
        }],
        /**
         * Left
         * @see https://tailwindcss.com/docs/top-right-bottom-left
         */
        left: [{
          left: scaleInset()
        }],
        /**
         * Visibility
         * @see https://tailwindcss.com/docs/visibility
         */
        visibility: ["visible", "invisible", "collapse"],
        /**
         * Z-Index
         * @see https://tailwindcss.com/docs/z-index
         */
        z: [{
          z: [isInteger, "auto", isArbitraryVariable, isArbitraryValue]
        }],
        // ------------------------
        // --- Flexbox and Grid ---
        // ------------------------
        /**
         * Flex Basis
         * @see https://tailwindcss.com/docs/flex-basis
         */
        basis: [{
          basis: [isFraction, "full", "auto", themeContainer, ...scaleUnambiguousSpacing()]
        }],
        /**
         * Flex Direction
         * @see https://tailwindcss.com/docs/flex-direction
         */
        "flex-direction": [{
          flex: ["row", "row-reverse", "col", "col-reverse"]
        }],
        /**
         * Flex Wrap
         * @see https://tailwindcss.com/docs/flex-wrap
         */
        "flex-wrap": [{
          flex: ["nowrap", "wrap", "wrap-reverse"]
        }],
        /**
         * Flex
         * @see https://tailwindcss.com/docs/flex
         */
        flex: [{
          flex: [isNumber, isFraction, "auto", "initial", "none", isArbitraryValue]
        }],
        /**
         * Flex Grow
         * @see https://tailwindcss.com/docs/flex-grow
         */
        grow: [{
          grow: ["", isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Flex Shrink
         * @see https://tailwindcss.com/docs/flex-shrink
         */
        shrink: [{
          shrink: ["", isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Order
         * @see https://tailwindcss.com/docs/order
         */
        order: [{
          order: [isInteger, "first", "last", "none", isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Grid Template Columns
         * @see https://tailwindcss.com/docs/grid-template-columns
         */
        "grid-cols": [{
          "grid-cols": scaleGridTemplateColsRows()
        }],
        /**
         * Grid Column Start / End
         * @see https://tailwindcss.com/docs/grid-column
         */
        "col-start-end": [{
          col: scaleGridColRowStartAndEnd()
        }],
        /**
         * Grid Column Start
         * @see https://tailwindcss.com/docs/grid-column
         */
        "col-start": [{
          "col-start": scaleGridColRowStartOrEnd()
        }],
        /**
         * Grid Column End
         * @see https://tailwindcss.com/docs/grid-column
         */
        "col-end": [{
          "col-end": scaleGridColRowStartOrEnd()
        }],
        /**
         * Grid Template Rows
         * @see https://tailwindcss.com/docs/grid-template-rows
         */
        "grid-rows": [{
          "grid-rows": scaleGridTemplateColsRows()
        }],
        /**
         * Grid Row Start / End
         * @see https://tailwindcss.com/docs/grid-row
         */
        "row-start-end": [{
          row: scaleGridColRowStartAndEnd()
        }],
        /**
         * Grid Row Start
         * @see https://tailwindcss.com/docs/grid-row
         */
        "row-start": [{
          "row-start": scaleGridColRowStartOrEnd()
        }],
        /**
         * Grid Row End
         * @see https://tailwindcss.com/docs/grid-row
         */
        "row-end": [{
          "row-end": scaleGridColRowStartOrEnd()
        }],
        /**
         * Grid Auto Flow
         * @see https://tailwindcss.com/docs/grid-auto-flow
         */
        "grid-flow": [{
          "grid-flow": ["row", "col", "dense", "row-dense", "col-dense"]
        }],
        /**
         * Grid Auto Columns
         * @see https://tailwindcss.com/docs/grid-auto-columns
         */
        "auto-cols": [{
          "auto-cols": scaleGridAutoColsRows()
        }],
        /**
         * Grid Auto Rows
         * @see https://tailwindcss.com/docs/grid-auto-rows
         */
        "auto-rows": [{
          "auto-rows": scaleGridAutoColsRows()
        }],
        /**
         * Gap
         * @see https://tailwindcss.com/docs/gap
         */
        gap: [{
          gap: scaleUnambiguousSpacing()
        }],
        /**
         * Gap X
         * @see https://tailwindcss.com/docs/gap
         */
        "gap-x": [{
          "gap-x": scaleUnambiguousSpacing()
        }],
        /**
         * Gap Y
         * @see https://tailwindcss.com/docs/gap
         */
        "gap-y": [{
          "gap-y": scaleUnambiguousSpacing()
        }],
        /**
         * Justify Content
         * @see https://tailwindcss.com/docs/justify-content
         */
        "justify-content": [{
          justify: [...scaleAlignPrimaryAxis(), "normal"]
        }],
        /**
         * Justify Items
         * @see https://tailwindcss.com/docs/justify-items
         */
        "justify-items": [{
          "justify-items": [...scaleAlignSecondaryAxis(), "normal"]
        }],
        /**
         * Justify Self
         * @see https://tailwindcss.com/docs/justify-self
         */
        "justify-self": [{
          "justify-self": ["auto", ...scaleAlignSecondaryAxis()]
        }],
        /**
         * Align Content
         * @see https://tailwindcss.com/docs/align-content
         */
        "align-content": [{
          content: ["normal", ...scaleAlignPrimaryAxis()]
        }],
        /**
         * Align Items
         * @see https://tailwindcss.com/docs/align-items
         */
        "align-items": [{
          items: [...scaleAlignSecondaryAxis(), {
            baseline: ["", "last"]
          }]
        }],
        /**
         * Align Self
         * @see https://tailwindcss.com/docs/align-self
         */
        "align-self": [{
          self: ["auto", ...scaleAlignSecondaryAxis(), {
            baseline: ["", "last"]
          }]
        }],
        /**
         * Place Content
         * @see https://tailwindcss.com/docs/place-content
         */
        "place-content": [{
          "place-content": scaleAlignPrimaryAxis()
        }],
        /**
         * Place Items
         * @see https://tailwindcss.com/docs/place-items
         */
        "place-items": [{
          "place-items": [...scaleAlignSecondaryAxis(), "baseline"]
        }],
        /**
         * Place Self
         * @see https://tailwindcss.com/docs/place-self
         */
        "place-self": [{
          "place-self": ["auto", ...scaleAlignSecondaryAxis()]
        }],
        // Spacing
        /**
         * Padding
         * @see https://tailwindcss.com/docs/padding
         */
        p: [{
          p: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Inline
         * @see https://tailwindcss.com/docs/padding
         */
        px: [{
          px: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Block
         * @see https://tailwindcss.com/docs/padding
         */
        py: [{
          py: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Inline Start
         * @see https://tailwindcss.com/docs/padding
         */
        ps: [{
          ps: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Inline End
         * @see https://tailwindcss.com/docs/padding
         */
        pe: [{
          pe: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Block Start
         * @see https://tailwindcss.com/docs/padding
         */
        pbs: [{
          pbs: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Block End
         * @see https://tailwindcss.com/docs/padding
         */
        pbe: [{
          pbe: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Top
         * @see https://tailwindcss.com/docs/padding
         */
        pt: [{
          pt: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Right
         * @see https://tailwindcss.com/docs/padding
         */
        pr: [{
          pr: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Bottom
         * @see https://tailwindcss.com/docs/padding
         */
        pb: [{
          pb: scaleUnambiguousSpacing()
        }],
        /**
         * Padding Left
         * @see https://tailwindcss.com/docs/padding
         */
        pl: [{
          pl: scaleUnambiguousSpacing()
        }],
        /**
         * Margin
         * @see https://tailwindcss.com/docs/margin
         */
        m: [{
          m: scaleMargin()
        }],
        /**
         * Margin Inline
         * @see https://tailwindcss.com/docs/margin
         */
        mx: [{
          mx: scaleMargin()
        }],
        /**
         * Margin Block
         * @see https://tailwindcss.com/docs/margin
         */
        my: [{
          my: scaleMargin()
        }],
        /**
         * Margin Inline Start
         * @see https://tailwindcss.com/docs/margin
         */
        ms: [{
          ms: scaleMargin()
        }],
        /**
         * Margin Inline End
         * @see https://tailwindcss.com/docs/margin
         */
        me: [{
          me: scaleMargin()
        }],
        /**
         * Margin Block Start
         * @see https://tailwindcss.com/docs/margin
         */
        mbs: [{
          mbs: scaleMargin()
        }],
        /**
         * Margin Block End
         * @see https://tailwindcss.com/docs/margin
         */
        mbe: [{
          mbe: scaleMargin()
        }],
        /**
         * Margin Top
         * @see https://tailwindcss.com/docs/margin
         */
        mt: [{
          mt: scaleMargin()
        }],
        /**
         * Margin Right
         * @see https://tailwindcss.com/docs/margin
         */
        mr: [{
          mr: scaleMargin()
        }],
        /**
         * Margin Bottom
         * @see https://tailwindcss.com/docs/margin
         */
        mb: [{
          mb: scaleMargin()
        }],
        /**
         * Margin Left
         * @see https://tailwindcss.com/docs/margin
         */
        ml: [{
          ml: scaleMargin()
        }],
        /**
         * Space Between X
         * @see https://tailwindcss.com/docs/margin#adding-space-between-children
         */
        "space-x": [{
          "space-x": scaleUnambiguousSpacing()
        }],
        /**
         * Space Between X Reverse
         * @see https://tailwindcss.com/docs/margin#adding-space-between-children
         */
        "space-x-reverse": ["space-x-reverse"],
        /**
         * Space Between Y
         * @see https://tailwindcss.com/docs/margin#adding-space-between-children
         */
        "space-y": [{
          "space-y": scaleUnambiguousSpacing()
        }],
        /**
         * Space Between Y Reverse
         * @see https://tailwindcss.com/docs/margin#adding-space-between-children
         */
        "space-y-reverse": ["space-y-reverse"],
        // --------------
        // --- Sizing ---
        // --------------
        /**
         * Size
         * @see https://tailwindcss.com/docs/width#setting-both-width-and-height
         */
        size: [{
          size: scaleSizing()
        }],
        /**
         * Inline Size
         * @see https://tailwindcss.com/docs/width
         */
        "inline-size": [{
          inline: ["auto", ...scaleSizingInline()]
        }],
        /**
         * Min-Inline Size
         * @see https://tailwindcss.com/docs/min-width
         */
        "min-inline-size": [{
          "min-inline": ["auto", ...scaleSizingInline()]
        }],
        /**
         * Max-Inline Size
         * @see https://tailwindcss.com/docs/max-width
         */
        "max-inline-size": [{
          "max-inline": ["none", ...scaleSizingInline()]
        }],
        /**
         * Block Size
         * @see https://tailwindcss.com/docs/height
         */
        "block-size": [{
          block: ["auto", ...scaleSizingBlock()]
        }],
        /**
         * Min-Block Size
         * @see https://tailwindcss.com/docs/min-height
         */
        "min-block-size": [{
          "min-block": ["auto", ...scaleSizingBlock()]
        }],
        /**
         * Max-Block Size
         * @see https://tailwindcss.com/docs/max-height
         */
        "max-block-size": [{
          "max-block": ["none", ...scaleSizingBlock()]
        }],
        /**
         * Width
         * @see https://tailwindcss.com/docs/width
         */
        w: [{
          w: [themeContainer, "screen", ...scaleSizing()]
        }],
        /**
         * Min-Width
         * @see https://tailwindcss.com/docs/min-width
         */
        "min-w": [{
          "min-w": [
            themeContainer,
            "screen",
            /** Deprecated. @see https://github.com/tailwindlabs/tailwindcss.com/issues/2027#issuecomment-2620152757 */
            "none",
            ...scaleSizing()
          ]
        }],
        /**
         * Max-Width
         * @see https://tailwindcss.com/docs/max-width
         */
        "max-w": [{
          "max-w": [
            themeContainer,
            "screen",
            "none",
            /** Deprecated since Tailwind CSS v4.0.0. @see https://github.com/tailwindlabs/tailwindcss.com/issues/2027#issuecomment-2620152757 */
            "prose",
            /** Deprecated since Tailwind CSS v4.0.0. @see https://github.com/tailwindlabs/tailwindcss.com/issues/2027#issuecomment-2620152757 */
            {
              screen: [themeBreakpoint]
            },
            ...scaleSizing()
          ]
        }],
        /**
         * Height
         * @see https://tailwindcss.com/docs/height
         */
        h: [{
          h: ["screen", "lh", ...scaleSizing()]
        }],
        /**
         * Min-Height
         * @see https://tailwindcss.com/docs/min-height
         */
        "min-h": [{
          "min-h": ["screen", "lh", "none", ...scaleSizing()]
        }],
        /**
         * Max-Height
         * @see https://tailwindcss.com/docs/max-height
         */
        "max-h": [{
          "max-h": ["screen", "lh", ...scaleSizing()]
        }],
        // ------------------
        // --- Typography ---
        // ------------------
        /**
         * Font Size
         * @see https://tailwindcss.com/docs/font-size
         */
        "font-size": [{
          text: ["base", themeText, isArbitraryVariableLength, isArbitraryLength]
        }],
        /**
         * Font Smoothing
         * @see https://tailwindcss.com/docs/font-smoothing
         */
        "font-smoothing": ["antialiased", "subpixel-antialiased"],
        /**
         * Font Style
         * @see https://tailwindcss.com/docs/font-style
         */
        "font-style": ["italic", "not-italic"],
        /**
         * Font Weight
         * @see https://tailwindcss.com/docs/font-weight
         */
        "font-weight": [{
          font: [themeFontWeight, isArbitraryVariableWeight, isArbitraryWeight]
        }],
        /**
         * Font Stretch
         * @see https://tailwindcss.com/docs/font-stretch
         */
        "font-stretch": [{
          "font-stretch": ["ultra-condensed", "extra-condensed", "condensed", "semi-condensed", "normal", "semi-expanded", "expanded", "extra-expanded", "ultra-expanded", isPercent, isArbitraryValue]
        }],
        /**
         * Font Family
         * @see https://tailwindcss.com/docs/font-family
         */
        "font-family": [{
          font: [isArbitraryVariableFamilyName, isArbitraryFamilyName, themeFont]
        }],
        /**
         * Font Feature Settings
         * @see https://tailwindcss.com/docs/font-feature-settings
         */
        "font-features": [{
          "font-features": [isArbitraryValue]
        }],
        /**
         * Font Variant Numeric
         * @see https://tailwindcss.com/docs/font-variant-numeric
         */
        "fvn-normal": ["normal-nums"],
        /**
         * Font Variant Numeric
         * @see https://tailwindcss.com/docs/font-variant-numeric
         */
        "fvn-ordinal": ["ordinal"],
        /**
         * Font Variant Numeric
         * @see https://tailwindcss.com/docs/font-variant-numeric
         */
        "fvn-slashed-zero": ["slashed-zero"],
        /**
         * Font Variant Numeric
         * @see https://tailwindcss.com/docs/font-variant-numeric
         */
        "fvn-figure": ["lining-nums", "oldstyle-nums"],
        /**
         * Font Variant Numeric
         * @see https://tailwindcss.com/docs/font-variant-numeric
         */
        "fvn-spacing": ["proportional-nums", "tabular-nums"],
        /**
         * Font Variant Numeric
         * @see https://tailwindcss.com/docs/font-variant-numeric
         */
        "fvn-fraction": ["diagonal-fractions", "stacked-fractions"],
        /**
         * Letter Spacing
         * @see https://tailwindcss.com/docs/letter-spacing
         */
        tracking: [{
          tracking: [themeTracking, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Line Clamp
         * @see https://tailwindcss.com/docs/line-clamp
         */
        "line-clamp": [{
          "line-clamp": [isNumber, "none", isArbitraryVariable, isArbitraryNumber]
        }],
        /**
         * Line Height
         * @see https://tailwindcss.com/docs/line-height
         */
        leading: [{
          leading: [
            /** Deprecated since Tailwind CSS v4.0.0. @see https://github.com/tailwindlabs/tailwindcss.com/issues/2027#issuecomment-2620152757 */
            themeLeading,
            ...scaleUnambiguousSpacing()
          ]
        }],
        /**
         * List Style Image
         * @see https://tailwindcss.com/docs/list-style-image
         */
        "list-image": [{
          "list-image": ["none", isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * List Style Position
         * @see https://tailwindcss.com/docs/list-style-position
         */
        "list-style-position": [{
          list: ["inside", "outside"]
        }],
        /**
         * List Style Type
         * @see https://tailwindcss.com/docs/list-style-type
         */
        "list-style-type": [{
          list: ["disc", "decimal", "none", isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Text Alignment
         * @see https://tailwindcss.com/docs/text-align
         */
        "text-alignment": [{
          text: ["left", "center", "right", "justify", "start", "end"]
        }],
        /**
         * Placeholder Color
         * @deprecated since Tailwind CSS v3.0.0
         * @see https://v3.tailwindcss.com/docs/placeholder-color
         */
        "placeholder-color": [{
          placeholder: scaleColor()
        }],
        /**
         * Text Color
         * @see https://tailwindcss.com/docs/text-color
         */
        "text-color": [{
          text: scaleColor()
        }],
        /**
         * Text Decoration
         * @see https://tailwindcss.com/docs/text-decoration
         */
        "text-decoration": ["underline", "overline", "line-through", "no-underline"],
        /**
         * Text Decoration Style
         * @see https://tailwindcss.com/docs/text-decoration-style
         */
        "text-decoration-style": [{
          decoration: [...scaleLineStyle(), "wavy"]
        }],
        /**
         * Text Decoration Thickness
         * @see https://tailwindcss.com/docs/text-decoration-thickness
         */
        "text-decoration-thickness": [{
          decoration: [isNumber, "from-font", "auto", isArbitraryVariable, isArbitraryLength]
        }],
        /**
         * Text Decoration Color
         * @see https://tailwindcss.com/docs/text-decoration-color
         */
        "text-decoration-color": [{
          decoration: scaleColor()
        }],
        /**
         * Text Underline Offset
         * @see https://tailwindcss.com/docs/text-underline-offset
         */
        "underline-offset": [{
          "underline-offset": [isNumber, "auto", isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Text Transform
         * @see https://tailwindcss.com/docs/text-transform
         */
        "text-transform": ["uppercase", "lowercase", "capitalize", "normal-case"],
        /**
         * Text Overflow
         * @see https://tailwindcss.com/docs/text-overflow
         */
        "text-overflow": ["truncate", "text-ellipsis", "text-clip"],
        /**
         * Text Wrap
         * @see https://tailwindcss.com/docs/text-wrap
         */
        "text-wrap": [{
          text: ["wrap", "nowrap", "balance", "pretty"]
        }],
        /**
         * Text Indent
         * @see https://tailwindcss.com/docs/text-indent
         */
        indent: [{
          indent: scaleUnambiguousSpacing()
        }],
        /**
         * Vertical Alignment
         * @see https://tailwindcss.com/docs/vertical-align
         */
        "vertical-align": [{
          align: ["baseline", "top", "middle", "bottom", "text-top", "text-bottom", "sub", "super", isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Whitespace
         * @see https://tailwindcss.com/docs/whitespace
         */
        whitespace: [{
          whitespace: ["normal", "nowrap", "pre", "pre-line", "pre-wrap", "break-spaces"]
        }],
        /**
         * Word Break
         * @see https://tailwindcss.com/docs/word-break
         */
        break: [{
          break: ["normal", "words", "all", "keep"]
        }],
        /**
         * Overflow Wrap
         * @see https://tailwindcss.com/docs/overflow-wrap
         */
        wrap: [{
          wrap: ["break-word", "anywhere", "normal"]
        }],
        /**
         * Hyphens
         * @see https://tailwindcss.com/docs/hyphens
         */
        hyphens: [{
          hyphens: ["none", "manual", "auto"]
        }],
        /**
         * Content
         * @see https://tailwindcss.com/docs/content
         */
        content: [{
          content: ["none", isArbitraryVariable, isArbitraryValue]
        }],
        // -------------------
        // --- Backgrounds ---
        // -------------------
        /**
         * Background Attachment
         * @see https://tailwindcss.com/docs/background-attachment
         */
        "bg-attachment": [{
          bg: ["fixed", "local", "scroll"]
        }],
        /**
         * Background Clip
         * @see https://tailwindcss.com/docs/background-clip
         */
        "bg-clip": [{
          "bg-clip": ["border", "padding", "content", "text"]
        }],
        /**
         * Background Origin
         * @see https://tailwindcss.com/docs/background-origin
         */
        "bg-origin": [{
          "bg-origin": ["border", "padding", "content"]
        }],
        /**
         * Background Position
         * @see https://tailwindcss.com/docs/background-position
         */
        "bg-position": [{
          bg: scaleBgPosition()
        }],
        /**
         * Background Repeat
         * @see https://tailwindcss.com/docs/background-repeat
         */
        "bg-repeat": [{
          bg: scaleBgRepeat()
        }],
        /**
         * Background Size
         * @see https://tailwindcss.com/docs/background-size
         */
        "bg-size": [{
          bg: scaleBgSize()
        }],
        /**
         * Background Image
         * @see https://tailwindcss.com/docs/background-image
         */
        "bg-image": [{
          bg: ["none", {
            linear: [{
              to: ["t", "tr", "r", "br", "b", "bl", "l", "tl"]
            }, isInteger, isArbitraryVariable, isArbitraryValue],
            radial: ["", isArbitraryVariable, isArbitraryValue],
            conic: [isInteger, isArbitraryVariable, isArbitraryValue]
          }, isArbitraryVariableImage, isArbitraryImage]
        }],
        /**
         * Background Color
         * @see https://tailwindcss.com/docs/background-color
         */
        "bg-color": [{
          bg: scaleColor()
        }],
        /**
         * Gradient Color Stops From Position
         * @see https://tailwindcss.com/docs/gradient-color-stops
         */
        "gradient-from-pos": [{
          from: scaleGradientStopPosition()
        }],
        /**
         * Gradient Color Stops Via Position
         * @see https://tailwindcss.com/docs/gradient-color-stops
         */
        "gradient-via-pos": [{
          via: scaleGradientStopPosition()
        }],
        /**
         * Gradient Color Stops To Position
         * @see https://tailwindcss.com/docs/gradient-color-stops
         */
        "gradient-to-pos": [{
          to: scaleGradientStopPosition()
        }],
        /**
         * Gradient Color Stops From
         * @see https://tailwindcss.com/docs/gradient-color-stops
         */
        "gradient-from": [{
          from: scaleColor()
        }],
        /**
         * Gradient Color Stops Via
         * @see https://tailwindcss.com/docs/gradient-color-stops
         */
        "gradient-via": [{
          via: scaleColor()
        }],
        /**
         * Gradient Color Stops To
         * @see https://tailwindcss.com/docs/gradient-color-stops
         */
        "gradient-to": [{
          to: scaleColor()
        }],
        // ---------------
        // --- Borders ---
        // ---------------
        /**
         * Border Radius
         * @see https://tailwindcss.com/docs/border-radius
         */
        rounded: [{
          rounded: scaleRadius()
        }],
        /**
         * Border Radius Start
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-s": [{
          "rounded-s": scaleRadius()
        }],
        /**
         * Border Radius End
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-e": [{
          "rounded-e": scaleRadius()
        }],
        /**
         * Border Radius Top
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-t": [{
          "rounded-t": scaleRadius()
        }],
        /**
         * Border Radius Right
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-r": [{
          "rounded-r": scaleRadius()
        }],
        /**
         * Border Radius Bottom
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-b": [{
          "rounded-b": scaleRadius()
        }],
        /**
         * Border Radius Left
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-l": [{
          "rounded-l": scaleRadius()
        }],
        /**
         * Border Radius Start Start
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-ss": [{
          "rounded-ss": scaleRadius()
        }],
        /**
         * Border Radius Start End
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-se": [{
          "rounded-se": scaleRadius()
        }],
        /**
         * Border Radius End End
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-ee": [{
          "rounded-ee": scaleRadius()
        }],
        /**
         * Border Radius End Start
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-es": [{
          "rounded-es": scaleRadius()
        }],
        /**
         * Border Radius Top Left
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-tl": [{
          "rounded-tl": scaleRadius()
        }],
        /**
         * Border Radius Top Right
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-tr": [{
          "rounded-tr": scaleRadius()
        }],
        /**
         * Border Radius Bottom Right
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-br": [{
          "rounded-br": scaleRadius()
        }],
        /**
         * Border Radius Bottom Left
         * @see https://tailwindcss.com/docs/border-radius
         */
        "rounded-bl": [{
          "rounded-bl": scaleRadius()
        }],
        /**
         * Border Width
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w": [{
          border: scaleBorderWidth()
        }],
        /**
         * Border Width Inline
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-x": [{
          "border-x": scaleBorderWidth()
        }],
        /**
         * Border Width Block
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-y": [{
          "border-y": scaleBorderWidth()
        }],
        /**
         * Border Width Inline Start
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-s": [{
          "border-s": scaleBorderWidth()
        }],
        /**
         * Border Width Inline End
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-e": [{
          "border-e": scaleBorderWidth()
        }],
        /**
         * Border Width Block Start
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-bs": [{
          "border-bs": scaleBorderWidth()
        }],
        /**
         * Border Width Block End
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-be": [{
          "border-be": scaleBorderWidth()
        }],
        /**
         * Border Width Top
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-t": [{
          "border-t": scaleBorderWidth()
        }],
        /**
         * Border Width Right
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-r": [{
          "border-r": scaleBorderWidth()
        }],
        /**
         * Border Width Bottom
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-b": [{
          "border-b": scaleBorderWidth()
        }],
        /**
         * Border Width Left
         * @see https://tailwindcss.com/docs/border-width
         */
        "border-w-l": [{
          "border-l": scaleBorderWidth()
        }],
        /**
         * Divide Width X
         * @see https://tailwindcss.com/docs/border-width#between-children
         */
        "divide-x": [{
          "divide-x": scaleBorderWidth()
        }],
        /**
         * Divide Width X Reverse
         * @see https://tailwindcss.com/docs/border-width#between-children
         */
        "divide-x-reverse": ["divide-x-reverse"],
        /**
         * Divide Width Y
         * @see https://tailwindcss.com/docs/border-width#between-children
         */
        "divide-y": [{
          "divide-y": scaleBorderWidth()
        }],
        /**
         * Divide Width Y Reverse
         * @see https://tailwindcss.com/docs/border-width#between-children
         */
        "divide-y-reverse": ["divide-y-reverse"],
        /**
         * Border Style
         * @see https://tailwindcss.com/docs/border-style
         */
        "border-style": [{
          border: [...scaleLineStyle(), "hidden", "none"]
        }],
        /**
         * Divide Style
         * @see https://tailwindcss.com/docs/border-style#setting-the-divider-style
         */
        "divide-style": [{
          divide: [...scaleLineStyle(), "hidden", "none"]
        }],
        /**
         * Border Color
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color": [{
          border: scaleColor()
        }],
        /**
         * Border Color Inline
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-x": [{
          "border-x": scaleColor()
        }],
        /**
         * Border Color Block
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-y": [{
          "border-y": scaleColor()
        }],
        /**
         * Border Color Inline Start
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-s": [{
          "border-s": scaleColor()
        }],
        /**
         * Border Color Inline End
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-e": [{
          "border-e": scaleColor()
        }],
        /**
         * Border Color Block Start
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-bs": [{
          "border-bs": scaleColor()
        }],
        /**
         * Border Color Block End
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-be": [{
          "border-be": scaleColor()
        }],
        /**
         * Border Color Top
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-t": [{
          "border-t": scaleColor()
        }],
        /**
         * Border Color Right
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-r": [{
          "border-r": scaleColor()
        }],
        /**
         * Border Color Bottom
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-b": [{
          "border-b": scaleColor()
        }],
        /**
         * Border Color Left
         * @see https://tailwindcss.com/docs/border-color
         */
        "border-color-l": [{
          "border-l": scaleColor()
        }],
        /**
         * Divide Color
         * @see https://tailwindcss.com/docs/divide-color
         */
        "divide-color": [{
          divide: scaleColor()
        }],
        /**
         * Outline Style
         * @see https://tailwindcss.com/docs/outline-style
         */
        "outline-style": [{
          outline: [...scaleLineStyle(), "none", "hidden"]
        }],
        /**
         * Outline Offset
         * @see https://tailwindcss.com/docs/outline-offset
         */
        "outline-offset": [{
          "outline-offset": [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Outline Width
         * @see https://tailwindcss.com/docs/outline-width
         */
        "outline-w": [{
          outline: ["", isNumber, isArbitraryVariableLength, isArbitraryLength]
        }],
        /**
         * Outline Color
         * @see https://tailwindcss.com/docs/outline-color
         */
        "outline-color": [{
          outline: scaleColor()
        }],
        // ---------------
        // --- Effects ---
        // ---------------
        /**
         * Box Shadow
         * @see https://tailwindcss.com/docs/box-shadow
         */
        shadow: [{
          shadow: [
            // Deprecated since Tailwind CSS v4.0.0
            "",
            "none",
            themeShadow,
            isArbitraryVariableShadow,
            isArbitraryShadow
          ]
        }],
        /**
         * Box Shadow Color
         * @see https://tailwindcss.com/docs/box-shadow#setting-the-shadow-color
         */
        "shadow-color": [{
          shadow: scaleColor()
        }],
        /**
         * Inset Box Shadow
         * @see https://tailwindcss.com/docs/box-shadow#adding-an-inset-shadow
         */
        "inset-shadow": [{
          "inset-shadow": ["none", themeInsetShadow, isArbitraryVariableShadow, isArbitraryShadow]
        }],
        /**
         * Inset Box Shadow Color
         * @see https://tailwindcss.com/docs/box-shadow#setting-the-inset-shadow-color
         */
        "inset-shadow-color": [{
          "inset-shadow": scaleColor()
        }],
        /**
         * Ring Width
         * @see https://tailwindcss.com/docs/box-shadow#adding-a-ring
         */
        "ring-w": [{
          ring: scaleBorderWidth()
        }],
        /**
         * Ring Width Inset
         * @see https://v3.tailwindcss.com/docs/ring-width#inset-rings
         * @deprecated since Tailwind CSS v4.0.0
         * @see https://github.com/tailwindlabs/tailwindcss/blob/v4.0.0/packages/tailwindcss/src/utilities.ts#L4158
         */
        "ring-w-inset": ["ring-inset"],
        /**
         * Ring Color
         * @see https://tailwindcss.com/docs/box-shadow#setting-the-ring-color
         */
        "ring-color": [{
          ring: scaleColor()
        }],
        /**
         * Ring Offset Width
         * @see https://v3.tailwindcss.com/docs/ring-offset-width
         * @deprecated since Tailwind CSS v4.0.0
         * @see https://github.com/tailwindlabs/tailwindcss/blob/v4.0.0/packages/tailwindcss/src/utilities.ts#L4158
         */
        "ring-offset-w": [{
          "ring-offset": [isNumber, isArbitraryLength]
        }],
        /**
         * Ring Offset Color
         * @see https://v3.tailwindcss.com/docs/ring-offset-color
         * @deprecated since Tailwind CSS v4.0.0
         * @see https://github.com/tailwindlabs/tailwindcss/blob/v4.0.0/packages/tailwindcss/src/utilities.ts#L4158
         */
        "ring-offset-color": [{
          "ring-offset": scaleColor()
        }],
        /**
         * Inset Ring Width
         * @see https://tailwindcss.com/docs/box-shadow#adding-an-inset-ring
         */
        "inset-ring-w": [{
          "inset-ring": scaleBorderWidth()
        }],
        /**
         * Inset Ring Color
         * @see https://tailwindcss.com/docs/box-shadow#setting-the-inset-ring-color
         */
        "inset-ring-color": [{
          "inset-ring": scaleColor()
        }],
        /**
         * Text Shadow
         * @see https://tailwindcss.com/docs/text-shadow
         */
        "text-shadow": [{
          "text-shadow": ["none", themeTextShadow, isArbitraryVariableShadow, isArbitraryShadow]
        }],
        /**
         * Text Shadow Color
         * @see https://tailwindcss.com/docs/text-shadow#setting-the-shadow-color
         */
        "text-shadow-color": [{
          "text-shadow": scaleColor()
        }],
        /**
         * Opacity
         * @see https://tailwindcss.com/docs/opacity
         */
        opacity: [{
          opacity: [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Mix Blend Mode
         * @see https://tailwindcss.com/docs/mix-blend-mode
         */
        "mix-blend": [{
          "mix-blend": [...scaleBlendMode(), "plus-darker", "plus-lighter"]
        }],
        /**
         * Background Blend Mode
         * @see https://tailwindcss.com/docs/background-blend-mode
         */
        "bg-blend": [{
          "bg-blend": scaleBlendMode()
        }],
        /**
         * Mask Clip
         * @see https://tailwindcss.com/docs/mask-clip
         */
        "mask-clip": [{
          "mask-clip": ["border", "padding", "content", "fill", "stroke", "view"]
        }, "mask-no-clip"],
        /**
         * Mask Composite
         * @see https://tailwindcss.com/docs/mask-composite
         */
        "mask-composite": [{
          mask: ["add", "subtract", "intersect", "exclude"]
        }],
        /**
         * Mask Image
         * @see https://tailwindcss.com/docs/mask-image
         */
        "mask-image-linear-pos": [{
          "mask-linear": [isNumber]
        }],
        "mask-image-linear-from-pos": [{
          "mask-linear-from": scaleMaskImagePosition()
        }],
        "mask-image-linear-to-pos": [{
          "mask-linear-to": scaleMaskImagePosition()
        }],
        "mask-image-linear-from-color": [{
          "mask-linear-from": scaleColor()
        }],
        "mask-image-linear-to-color": [{
          "mask-linear-to": scaleColor()
        }],
        "mask-image-t-from-pos": [{
          "mask-t-from": scaleMaskImagePosition()
        }],
        "mask-image-t-to-pos": [{
          "mask-t-to": scaleMaskImagePosition()
        }],
        "mask-image-t-from-color": [{
          "mask-t-from": scaleColor()
        }],
        "mask-image-t-to-color": [{
          "mask-t-to": scaleColor()
        }],
        "mask-image-r-from-pos": [{
          "mask-r-from": scaleMaskImagePosition()
        }],
        "mask-image-r-to-pos": [{
          "mask-r-to": scaleMaskImagePosition()
        }],
        "mask-image-r-from-color": [{
          "mask-r-from": scaleColor()
        }],
        "mask-image-r-to-color": [{
          "mask-r-to": scaleColor()
        }],
        "mask-image-b-from-pos": [{
          "mask-b-from": scaleMaskImagePosition()
        }],
        "mask-image-b-to-pos": [{
          "mask-b-to": scaleMaskImagePosition()
        }],
        "mask-image-b-from-color": [{
          "mask-b-from": scaleColor()
        }],
        "mask-image-b-to-color": [{
          "mask-b-to": scaleColor()
        }],
        "mask-image-l-from-pos": [{
          "mask-l-from": scaleMaskImagePosition()
        }],
        "mask-image-l-to-pos": [{
          "mask-l-to": scaleMaskImagePosition()
        }],
        "mask-image-l-from-color": [{
          "mask-l-from": scaleColor()
        }],
        "mask-image-l-to-color": [{
          "mask-l-to": scaleColor()
        }],
        "mask-image-x-from-pos": [{
          "mask-x-from": scaleMaskImagePosition()
        }],
        "mask-image-x-to-pos": [{
          "mask-x-to": scaleMaskImagePosition()
        }],
        "mask-image-x-from-color": [{
          "mask-x-from": scaleColor()
        }],
        "mask-image-x-to-color": [{
          "mask-x-to": scaleColor()
        }],
        "mask-image-y-from-pos": [{
          "mask-y-from": scaleMaskImagePosition()
        }],
        "mask-image-y-to-pos": [{
          "mask-y-to": scaleMaskImagePosition()
        }],
        "mask-image-y-from-color": [{
          "mask-y-from": scaleColor()
        }],
        "mask-image-y-to-color": [{
          "mask-y-to": scaleColor()
        }],
        "mask-image-radial": [{
          "mask-radial": [isArbitraryVariable, isArbitraryValue]
        }],
        "mask-image-radial-from-pos": [{
          "mask-radial-from": scaleMaskImagePosition()
        }],
        "mask-image-radial-to-pos": [{
          "mask-radial-to": scaleMaskImagePosition()
        }],
        "mask-image-radial-from-color": [{
          "mask-radial-from": scaleColor()
        }],
        "mask-image-radial-to-color": [{
          "mask-radial-to": scaleColor()
        }],
        "mask-image-radial-shape": [{
          "mask-radial": ["circle", "ellipse"]
        }],
        "mask-image-radial-size": [{
          "mask-radial": [{
            closest: ["side", "corner"],
            farthest: ["side", "corner"]
          }]
        }],
        "mask-image-radial-pos": [{
          "mask-radial-at": scalePosition()
        }],
        "mask-image-conic-pos": [{
          "mask-conic": [isNumber]
        }],
        "mask-image-conic-from-pos": [{
          "mask-conic-from": scaleMaskImagePosition()
        }],
        "mask-image-conic-to-pos": [{
          "mask-conic-to": scaleMaskImagePosition()
        }],
        "mask-image-conic-from-color": [{
          "mask-conic-from": scaleColor()
        }],
        "mask-image-conic-to-color": [{
          "mask-conic-to": scaleColor()
        }],
        /**
         * Mask Mode
         * @see https://tailwindcss.com/docs/mask-mode
         */
        "mask-mode": [{
          mask: ["alpha", "luminance", "match"]
        }],
        /**
         * Mask Origin
         * @see https://tailwindcss.com/docs/mask-origin
         */
        "mask-origin": [{
          "mask-origin": ["border", "padding", "content", "fill", "stroke", "view"]
        }],
        /**
         * Mask Position
         * @see https://tailwindcss.com/docs/mask-position
         */
        "mask-position": [{
          mask: scaleBgPosition()
        }],
        /**
         * Mask Repeat
         * @see https://tailwindcss.com/docs/mask-repeat
         */
        "mask-repeat": [{
          mask: scaleBgRepeat()
        }],
        /**
         * Mask Size
         * @see https://tailwindcss.com/docs/mask-size
         */
        "mask-size": [{
          mask: scaleBgSize()
        }],
        /**
         * Mask Type
         * @see https://tailwindcss.com/docs/mask-type
         */
        "mask-type": [{
          "mask-type": ["alpha", "luminance"]
        }],
        /**
         * Mask Image
         * @see https://tailwindcss.com/docs/mask-image
         */
        "mask-image": [{
          mask: ["none", isArbitraryVariable, isArbitraryValue]
        }],
        // ---------------
        // --- Filters ---
        // ---------------
        /**
         * Filter
         * @see https://tailwindcss.com/docs/filter
         */
        filter: [{
          filter: [
            // Deprecated since Tailwind CSS v3.0.0
            "",
            "none",
            isArbitraryVariable,
            isArbitraryValue
          ]
        }],
        /**
         * Blur
         * @see https://tailwindcss.com/docs/blur
         */
        blur: [{
          blur: scaleBlur()
        }],
        /**
         * Brightness
         * @see https://tailwindcss.com/docs/brightness
         */
        brightness: [{
          brightness: [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Contrast
         * @see https://tailwindcss.com/docs/contrast
         */
        contrast: [{
          contrast: [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Drop Shadow
         * @see https://tailwindcss.com/docs/drop-shadow
         */
        "drop-shadow": [{
          "drop-shadow": [
            // Deprecated since Tailwind CSS v4.0.0
            "",
            "none",
            themeDropShadow,
            isArbitraryVariableShadow,
            isArbitraryShadow
          ]
        }],
        /**
         * Drop Shadow Color
         * @see https://tailwindcss.com/docs/filter-drop-shadow#setting-the-shadow-color
         */
        "drop-shadow-color": [{
          "drop-shadow": scaleColor()
        }],
        /**
         * Grayscale
         * @see https://tailwindcss.com/docs/grayscale
         */
        grayscale: [{
          grayscale: ["", isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Hue Rotate
         * @see https://tailwindcss.com/docs/hue-rotate
         */
        "hue-rotate": [{
          "hue-rotate": [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Invert
         * @see https://tailwindcss.com/docs/invert
         */
        invert: [{
          invert: ["", isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Saturate
         * @see https://tailwindcss.com/docs/saturate
         */
        saturate: [{
          saturate: [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Sepia
         * @see https://tailwindcss.com/docs/sepia
         */
        sepia: [{
          sepia: ["", isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Backdrop Filter
         * @see https://tailwindcss.com/docs/backdrop-filter
         */
        "backdrop-filter": [{
          "backdrop-filter": [
            // Deprecated since Tailwind CSS v3.0.0
            "",
            "none",
            isArbitraryVariable,
            isArbitraryValue
          ]
        }],
        /**
         * Backdrop Blur
         * @see https://tailwindcss.com/docs/backdrop-blur
         */
        "backdrop-blur": [{
          "backdrop-blur": scaleBlur()
        }],
        /**
         * Backdrop Brightness
         * @see https://tailwindcss.com/docs/backdrop-brightness
         */
        "backdrop-brightness": [{
          "backdrop-brightness": [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Backdrop Contrast
         * @see https://tailwindcss.com/docs/backdrop-contrast
         */
        "backdrop-contrast": [{
          "backdrop-contrast": [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Backdrop Grayscale
         * @see https://tailwindcss.com/docs/backdrop-grayscale
         */
        "backdrop-grayscale": [{
          "backdrop-grayscale": ["", isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Backdrop Hue Rotate
         * @see https://tailwindcss.com/docs/backdrop-hue-rotate
         */
        "backdrop-hue-rotate": [{
          "backdrop-hue-rotate": [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Backdrop Invert
         * @see https://tailwindcss.com/docs/backdrop-invert
         */
        "backdrop-invert": [{
          "backdrop-invert": ["", isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Backdrop Opacity
         * @see https://tailwindcss.com/docs/backdrop-opacity
         */
        "backdrop-opacity": [{
          "backdrop-opacity": [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Backdrop Saturate
         * @see https://tailwindcss.com/docs/backdrop-saturate
         */
        "backdrop-saturate": [{
          "backdrop-saturate": [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Backdrop Sepia
         * @see https://tailwindcss.com/docs/backdrop-sepia
         */
        "backdrop-sepia": [{
          "backdrop-sepia": ["", isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        // --------------
        // --- Tables ---
        // --------------
        /**
         * Border Collapse
         * @see https://tailwindcss.com/docs/border-collapse
         */
        "border-collapse": [{
          border: ["collapse", "separate"]
        }],
        /**
         * Border Spacing
         * @see https://tailwindcss.com/docs/border-spacing
         */
        "border-spacing": [{
          "border-spacing": scaleUnambiguousSpacing()
        }],
        /**
         * Border Spacing X
         * @see https://tailwindcss.com/docs/border-spacing
         */
        "border-spacing-x": [{
          "border-spacing-x": scaleUnambiguousSpacing()
        }],
        /**
         * Border Spacing Y
         * @see https://tailwindcss.com/docs/border-spacing
         */
        "border-spacing-y": [{
          "border-spacing-y": scaleUnambiguousSpacing()
        }],
        /**
         * Table Layout
         * @see https://tailwindcss.com/docs/table-layout
         */
        "table-layout": [{
          table: ["auto", "fixed"]
        }],
        /**
         * Caption Side
         * @see https://tailwindcss.com/docs/caption-side
         */
        caption: [{
          caption: ["top", "bottom"]
        }],
        // ---------------------------------
        // --- Transitions and Animation ---
        // ---------------------------------
        /**
         * Transition Property
         * @see https://tailwindcss.com/docs/transition-property
         */
        transition: [{
          transition: ["", "all", "colors", "opacity", "shadow", "transform", "none", isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Transition Behavior
         * @see https://tailwindcss.com/docs/transition-behavior
         */
        "transition-behavior": [{
          transition: ["normal", "discrete"]
        }],
        /**
         * Transition Duration
         * @see https://tailwindcss.com/docs/transition-duration
         */
        duration: [{
          duration: [isNumber, "initial", isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Transition Timing Function
         * @see https://tailwindcss.com/docs/transition-timing-function
         */
        ease: [{
          ease: ["linear", "initial", themeEase, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Transition Delay
         * @see https://tailwindcss.com/docs/transition-delay
         */
        delay: [{
          delay: [isNumber, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Animation
         * @see https://tailwindcss.com/docs/animation
         */
        animate: [{
          animate: ["none", themeAnimate, isArbitraryVariable, isArbitraryValue]
        }],
        // ------------------
        // --- Transforms ---
        // ------------------
        /**
         * Backface Visibility
         * @see https://tailwindcss.com/docs/backface-visibility
         */
        backface: [{
          backface: ["hidden", "visible"]
        }],
        /**
         * Perspective
         * @see https://tailwindcss.com/docs/perspective
         */
        perspective: [{
          perspective: [themePerspective, isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Perspective Origin
         * @see https://tailwindcss.com/docs/perspective-origin
         */
        "perspective-origin": [{
          "perspective-origin": scalePositionWithArbitrary()
        }],
        /**
         * Rotate
         * @see https://tailwindcss.com/docs/rotate
         */
        rotate: [{
          rotate: scaleRotate()
        }],
        /**
         * Rotate X
         * @see https://tailwindcss.com/docs/rotate
         */
        "rotate-x": [{
          "rotate-x": scaleRotate()
        }],
        /**
         * Rotate Y
         * @see https://tailwindcss.com/docs/rotate
         */
        "rotate-y": [{
          "rotate-y": scaleRotate()
        }],
        /**
         * Rotate Z
         * @see https://tailwindcss.com/docs/rotate
         */
        "rotate-z": [{
          "rotate-z": scaleRotate()
        }],
        /**
         * Scale
         * @see https://tailwindcss.com/docs/scale
         */
        scale: [{
          scale: scaleScale()
        }],
        /**
         * Scale X
         * @see https://tailwindcss.com/docs/scale
         */
        "scale-x": [{
          "scale-x": scaleScale()
        }],
        /**
         * Scale Y
         * @see https://tailwindcss.com/docs/scale
         */
        "scale-y": [{
          "scale-y": scaleScale()
        }],
        /**
         * Scale Z
         * @see https://tailwindcss.com/docs/scale
         */
        "scale-z": [{
          "scale-z": scaleScale()
        }],
        /**
         * Scale 3D
         * @see https://tailwindcss.com/docs/scale
         */
        "scale-3d": ["scale-3d"],
        /**
         * Skew
         * @see https://tailwindcss.com/docs/skew
         */
        skew: [{
          skew: scaleSkew()
        }],
        /**
         * Skew X
         * @see https://tailwindcss.com/docs/skew
         */
        "skew-x": [{
          "skew-x": scaleSkew()
        }],
        /**
         * Skew Y
         * @see https://tailwindcss.com/docs/skew
         */
        "skew-y": [{
          "skew-y": scaleSkew()
        }],
        /**
         * Transform
         * @see https://tailwindcss.com/docs/transform
         */
        transform: [{
          transform: [isArbitraryVariable, isArbitraryValue, "", "none", "gpu", "cpu"]
        }],
        /**
         * Transform Origin
         * @see https://tailwindcss.com/docs/transform-origin
         */
        "transform-origin": [{
          origin: scalePositionWithArbitrary()
        }],
        /**
         * Transform Style
         * @see https://tailwindcss.com/docs/transform-style
         */
        "transform-style": [{
          transform: ["3d", "flat"]
        }],
        /**
         * Translate
         * @see https://tailwindcss.com/docs/translate
         */
        translate: [{
          translate: scaleTranslate()
        }],
        /**
         * Translate X
         * @see https://tailwindcss.com/docs/translate
         */
        "translate-x": [{
          "translate-x": scaleTranslate()
        }],
        /**
         * Translate Y
         * @see https://tailwindcss.com/docs/translate
         */
        "translate-y": [{
          "translate-y": scaleTranslate()
        }],
        /**
         * Translate Z
         * @see https://tailwindcss.com/docs/translate
         */
        "translate-z": [{
          "translate-z": scaleTranslate()
        }],
        /**
         * Translate None
         * @see https://tailwindcss.com/docs/translate
         */
        "translate-none": ["translate-none"],
        // ---------------------
        // --- Interactivity ---
        // ---------------------
        /**
         * Accent Color
         * @see https://tailwindcss.com/docs/accent-color
         */
        accent: [{
          accent: scaleColor()
        }],
        /**
         * Appearance
         * @see https://tailwindcss.com/docs/appearance
         */
        appearance: [{
          appearance: ["none", "auto"]
        }],
        /**
         * Caret Color
         * @see https://tailwindcss.com/docs/just-in-time-mode#caret-color-utilities
         */
        "caret-color": [{
          caret: scaleColor()
        }],
        /**
         * Color Scheme
         * @see https://tailwindcss.com/docs/color-scheme
         */
        "color-scheme": [{
          scheme: ["normal", "dark", "light", "light-dark", "only-dark", "only-light"]
        }],
        /**
         * Cursor
         * @see https://tailwindcss.com/docs/cursor
         */
        cursor: [{
          cursor: ["auto", "default", "pointer", "wait", "text", "move", "help", "not-allowed", "none", "context-menu", "progress", "cell", "crosshair", "vertical-text", "alias", "copy", "no-drop", "grab", "grabbing", "all-scroll", "col-resize", "row-resize", "n-resize", "e-resize", "s-resize", "w-resize", "ne-resize", "nw-resize", "se-resize", "sw-resize", "ew-resize", "ns-resize", "nesw-resize", "nwse-resize", "zoom-in", "zoom-out", isArbitraryVariable, isArbitraryValue]
        }],
        /**
         * Field Sizing
         * @see https://tailwindcss.com/docs/field-sizing
         */
        "field-sizing": [{
          "field-sizing": ["fixed", "content"]
        }],
        /**
         * Pointer Events
         * @see https://tailwindcss.com/docs/pointer-events
         */
        "pointer-events": [{
          "pointer-events": ["auto", "none"]
        }],
        /**
         * Resize
         * @see https://tailwindcss.com/docs/resize
         */
        resize: [{
          resize: ["none", "", "y", "x"]
        }],
        /**
         * Scroll Behavior
         * @see https://tailwindcss.com/docs/scroll-behavior
         */
        "scroll-behavior": [{
          scroll: ["auto", "smooth"]
        }],
        /**
         * Scroll Margin
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-m": [{
          "scroll-m": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Inline
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-mx": [{
          "scroll-mx": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Block
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-my": [{
          "scroll-my": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Inline Start
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-ms": [{
          "scroll-ms": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Inline End
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-me": [{
          "scroll-me": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Block Start
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-mbs": [{
          "scroll-mbs": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Block End
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-mbe": [{
          "scroll-mbe": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Top
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-mt": [{
          "scroll-mt": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Right
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-mr": [{
          "scroll-mr": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Bottom
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-mb": [{
          "scroll-mb": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Margin Left
         * @see https://tailwindcss.com/docs/scroll-margin
         */
        "scroll-ml": [{
          "scroll-ml": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-p": [{
          "scroll-p": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Inline
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-px": [{
          "scroll-px": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Block
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-py": [{
          "scroll-py": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Inline Start
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-ps": [{
          "scroll-ps": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Inline End
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-pe": [{
          "scroll-pe": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Block Start
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-pbs": [{
          "scroll-pbs": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Block End
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-pbe": [{
          "scroll-pbe": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Top
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-pt": [{
          "scroll-pt": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Right
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-pr": [{
          "scroll-pr": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Bottom
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-pb": [{
          "scroll-pb": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Padding Left
         * @see https://tailwindcss.com/docs/scroll-padding
         */
        "scroll-pl": [{
          "scroll-pl": scaleUnambiguousSpacing()
        }],
        /**
         * Scroll Snap Align
         * @see https://tailwindcss.com/docs/scroll-snap-align
         */
        "snap-align": [{
          snap: ["start", "end", "center", "align-none"]
        }],
        /**
         * Scroll Snap Stop
         * @see https://tailwindcss.com/docs/scroll-snap-stop
         */
        "snap-stop": [{
          snap: ["normal", "always"]
        }],
        /**
         * Scroll Snap Type
         * @see https://tailwindcss.com/docs/scroll-snap-type
         */
        "snap-type": [{
          snap: ["none", "x", "y", "both"]
        }],
        /**
         * Scroll Snap Type Strictness
         * @see https://tailwindcss.com/docs/scroll-snap-type
         */
        "snap-strictness": [{
          snap: ["mandatory", "proximity"]
        }],
        /**
         * Touch Action
         * @see https://tailwindcss.com/docs/touch-action
         */
        touch: [{
          touch: ["auto", "none", "manipulation"]
        }],
        /**
         * Touch Action X
         * @see https://tailwindcss.com/docs/touch-action
         */
        "touch-x": [{
          "touch-pan": ["x", "left", "right"]
        }],
        /**
         * Touch Action Y
         * @see https://tailwindcss.com/docs/touch-action
         */
        "touch-y": [{
          "touch-pan": ["y", "up", "down"]
        }],
        /**
         * Touch Action Pinch Zoom
         * @see https://tailwindcss.com/docs/touch-action
         */
        "touch-pz": ["touch-pinch-zoom"],
        /**
         * User Select
         * @see https://tailwindcss.com/docs/user-select
         */
        select: [{
          select: ["none", "text", "all", "auto"]
        }],
        /**
         * Will Change
         * @see https://tailwindcss.com/docs/will-change
         */
        "will-change": [{
          "will-change": ["auto", "scroll", "contents", "transform", isArbitraryVariable, isArbitraryValue]
        }],
        // -----------
        // --- SVG ---
        // -----------
        /**
         * Fill
         * @see https://tailwindcss.com/docs/fill
         */
        fill: [{
          fill: ["none", ...scaleColor()]
        }],
        /**
         * Stroke Width
         * @see https://tailwindcss.com/docs/stroke-width
         */
        "stroke-w": [{
          stroke: [isNumber, isArbitraryVariableLength, isArbitraryLength, isArbitraryNumber]
        }],
        /**
         * Stroke
         * @see https://tailwindcss.com/docs/stroke
         */
        stroke: [{
          stroke: ["none", ...scaleColor()]
        }],
        // ---------------------
        // --- Accessibility ---
        // ---------------------
        /**
         * Forced Color Adjust
         * @see https://tailwindcss.com/docs/forced-color-adjust
         */
        "forced-color-adjust": [{
          "forced-color-adjust": ["auto", "none"]
        }]
      },
      conflictingClassGroups: {
        overflow: ["overflow-x", "overflow-y"],
        overscroll: ["overscroll-x", "overscroll-y"],
        inset: ["inset-x", "inset-y", "inset-bs", "inset-be", "start", "end", "top", "right", "bottom", "left"],
        "inset-x": ["right", "left"],
        "inset-y": ["top", "bottom"],
        flex: ["basis", "grow", "shrink"],
        gap: ["gap-x", "gap-y"],
        p: ["px", "py", "ps", "pe", "pbs", "pbe", "pt", "pr", "pb", "pl"],
        px: ["pr", "pl"],
        py: ["pt", "pb"],
        m: ["mx", "my", "ms", "me", "mbs", "mbe", "mt", "mr", "mb", "ml"],
        mx: ["mr", "ml"],
        my: ["mt", "mb"],
        size: ["w", "h"],
        "font-size": ["leading"],
        "fvn-normal": ["fvn-ordinal", "fvn-slashed-zero", "fvn-figure", "fvn-spacing", "fvn-fraction"],
        "fvn-ordinal": ["fvn-normal"],
        "fvn-slashed-zero": ["fvn-normal"],
        "fvn-figure": ["fvn-normal"],
        "fvn-spacing": ["fvn-normal"],
        "fvn-fraction": ["fvn-normal"],
        "line-clamp": ["display", "overflow"],
        rounded: ["rounded-s", "rounded-e", "rounded-t", "rounded-r", "rounded-b", "rounded-l", "rounded-ss", "rounded-se", "rounded-ee", "rounded-es", "rounded-tl", "rounded-tr", "rounded-br", "rounded-bl"],
        "rounded-s": ["rounded-ss", "rounded-es"],
        "rounded-e": ["rounded-se", "rounded-ee"],
        "rounded-t": ["rounded-tl", "rounded-tr"],
        "rounded-r": ["rounded-tr", "rounded-br"],
        "rounded-b": ["rounded-br", "rounded-bl"],
        "rounded-l": ["rounded-tl", "rounded-bl"],
        "border-spacing": ["border-spacing-x", "border-spacing-y"],
        "border-w": ["border-w-x", "border-w-y", "border-w-s", "border-w-e", "border-w-bs", "border-w-be", "border-w-t", "border-w-r", "border-w-b", "border-w-l"],
        "border-w-x": ["border-w-r", "border-w-l"],
        "border-w-y": ["border-w-t", "border-w-b"],
        "border-color": ["border-color-x", "border-color-y", "border-color-s", "border-color-e", "border-color-bs", "border-color-be", "border-color-t", "border-color-r", "border-color-b", "border-color-l"],
        "border-color-x": ["border-color-r", "border-color-l"],
        "border-color-y": ["border-color-t", "border-color-b"],
        translate: ["translate-x", "translate-y", "translate-none"],
        "translate-none": ["translate", "translate-x", "translate-y", "translate-z"],
        "scroll-m": ["scroll-mx", "scroll-my", "scroll-ms", "scroll-me", "scroll-mbs", "scroll-mbe", "scroll-mt", "scroll-mr", "scroll-mb", "scroll-ml"],
        "scroll-mx": ["scroll-mr", "scroll-ml"],
        "scroll-my": ["scroll-mt", "scroll-mb"],
        "scroll-p": ["scroll-px", "scroll-py", "scroll-ps", "scroll-pe", "scroll-pbs", "scroll-pbe", "scroll-pt", "scroll-pr", "scroll-pb", "scroll-pl"],
        "scroll-px": ["scroll-pr", "scroll-pl"],
        "scroll-py": ["scroll-pt", "scroll-pb"],
        touch: ["touch-x", "touch-y", "touch-pz"],
        "touch-x": ["touch"],
        "touch-y": ["touch"],
        "touch-pz": ["touch"]
      },
      conflictingClassGroupModifiers: {
        "font-size": ["leading"]
      },
      orderSensitiveModifiers: ["*", "**", "after", "backdrop", "before", "details-content", "file", "first-letter", "first-line", "marker", "placeholder", "selection"]
    };
  };
  var twMerge = /* @__PURE__ */ createTailwindMerge(getDefaultConfig);

  // src/lib/utils.js
  function cn(...inputs) {
    return twMerge(clsx(inputs));
  }

  // src/components/ui/button.jsx
  var import_jsx_runtime2 = __toESM(require_jsx_runtime(), 1);
  var buttonVariants = cva("ui-button", {
    variants: {
      variant: {
        default: "ui-button--default",
        outline: "ui-button--outline",
        ghost: "ui-button--ghost",
        secondary: "ui-button--secondary",
        danger: "ui-button--danger"
      },
      size: {
        default: "ui-button--md",
        sm: "ui-button--sm",
        lg: "ui-button--lg",
        icon: "ui-button--icon"
      }
    },
    defaultVariants: {
      variant: "default",
      size: "default"
    }
  });
  function Button({ className, variant, size, asChild = false, ...props }) {
    const Comp = asChild ? Slot : "button";
    return /* @__PURE__ */ (0, import_jsx_runtime2.jsx)(Comp, { className: cn(buttonVariants({ variant, size }), className), ...props });
  }

  // src/components/ui/badge.jsx
  var import_react2 = __toESM(require_react(), 1);
  var import_jsx_runtime3 = __toESM(require_jsx_runtime(), 1);
  var badgeVariants = cva("ui-badge", {
    variants: {
      variant: {
        default: "ui-badge--default",
        outline: "ui-badge--outline",
        teal: "ui-badge--teal",
        amber: "ui-badge--amber",
        success: "ui-badge--success",
        warning: "ui-badge--warning",
        danger: "ui-badge--danger",
        muted: "ui-badge--muted"
      }
    },
    defaultVariants: {
      variant: "default"
    }
  });
  function Badge({ className, variant, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime3.jsx)("span", { className: cn(badgeVariants({ variant }), className), ...props });
  }

  // src/components/ui/card.jsx
  var import_react3 = __toESM(require_react(), 1);
  var import_jsx_runtime4 = __toESM(require_jsx_runtime(), 1);
  function Card({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime4.jsx)("div", { className: cn("ui-card panel", className), ...props });
  }
  function CardHeader({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime4.jsx)("div", { className: cn("ui-card__header", className), ...props });
  }
  function CardTitle({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime4.jsx)("h3", { className: cn("ui-card__title", className), ...props });
  }
  function CardDescription({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime4.jsx)("p", { className: cn("ui-card__description", className), ...props });
  }
  function CardContent({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime4.jsx)("div", { className: cn("ui-card__content", className), ...props });
  }

  // src/components/ui/input.jsx
  var import_react4 = __toESM(require_react(), 1);
  var import_jsx_runtime5 = __toESM(require_jsx_runtime(), 1);
  var Input = import_react4.default.forwardRef(function Input2({ className, ...props }, ref) {
    return /* @__PURE__ */ (0, import_jsx_runtime5.jsx)("input", { ref, className: cn("ui-input text-input", className), ...props });
  });

  // node_modules/@radix-ui/react-dialog/dist/index.mjs
  var React30 = __toESM(require_react(), 1);

  // node_modules/@radix-ui/primitive/dist/index.mjs
  var canUseDOM = !!(typeof window !== "undefined" && window.document && window.document.createElement);
  function composeEventHandlers(originalEventHandler, ourEventHandler, { checkForDefaultPrevented = true } = {}) {
    return function handleEvent(event) {
      originalEventHandler?.(event);
      if (checkForDefaultPrevented === false || !event.defaultPrevented) {
        return ourEventHandler?.(event);
      }
    };
  }

  // node_modules/@radix-ui/react-context/dist/index.mjs
  var React7 = __toESM(require_react(), 1);
  var import_jsx_runtime6 = __toESM(require_jsx_runtime(), 1);
  function createContext2(rootComponentName, defaultContext) {
    const Context = React7.createContext(defaultContext);
    const Provider = (props) => {
      const { children, ...context } = props;
      const value = React7.useMemo(() => context, Object.values(context));
      return /* @__PURE__ */ (0, import_jsx_runtime6.jsx)(Context.Provider, { value, children });
    };
    Provider.displayName = rootComponentName + "Provider";
    function useContext22(consumerName) {
      const context = React7.useContext(Context);
      if (context) return context;
      if (defaultContext !== void 0) return defaultContext;
      throw new Error(`\`${consumerName}\` must be used within \`${rootComponentName}\``);
    }
    return [Provider, useContext22];
  }
  function createContextScope(scopeName, createContextScopeDeps = []) {
    let defaultContexts = [];
    function createContext32(rootComponentName, defaultContext) {
      const BaseContext = React7.createContext(defaultContext);
      const index = defaultContexts.length;
      defaultContexts = [...defaultContexts, defaultContext];
      const Provider = (props) => {
        const { scope, children, ...context } = props;
        const Context = scope?.[scopeName]?.[index] || BaseContext;
        const value = React7.useMemo(() => context, Object.values(context));
        return /* @__PURE__ */ (0, import_jsx_runtime6.jsx)(Context.Provider, { value, children });
      };
      Provider.displayName = rootComponentName + "Provider";
      function useContext22(consumerName, scope) {
        const Context = scope?.[scopeName]?.[index] || BaseContext;
        const context = React7.useContext(Context);
        if (context) return context;
        if (defaultContext !== void 0) return defaultContext;
        throw new Error(`\`${consumerName}\` must be used within \`${rootComponentName}\``);
      }
      return [Provider, useContext22];
    }
    const createScope = () => {
      const scopeContexts = defaultContexts.map((defaultContext) => {
        return React7.createContext(defaultContext);
      });
      return function useScope(scope) {
        const contexts = scope?.[scopeName] || scopeContexts;
        return React7.useMemo(
          () => ({ [`__scope${scopeName}`]: { ...scope, [scopeName]: contexts } }),
          [scope, contexts]
        );
      };
    };
    createScope.scopeName = scopeName;
    return [createContext32, composeContextScopes(createScope, ...createContextScopeDeps)];
  }
  function composeContextScopes(...scopes) {
    const baseScope = scopes[0];
    if (scopes.length === 1) return baseScope;
    const createScope = () => {
      const scopeHooks = scopes.map((createScope2) => ({
        useScope: createScope2(),
        scopeName: createScope2.scopeName
      }));
      return function useComposedScopes(overrideScopes) {
        const nextScopes = scopeHooks.reduce((nextScopes2, { useScope, scopeName }) => {
          const scopeProps = useScope(overrideScopes);
          const currentScope = scopeProps[`__scope${scopeName}`];
          return { ...nextScopes2, ...currentScope };
        }, {});
        return React7.useMemo(() => ({ [`__scope${baseScope.scopeName}`]: nextScopes }), [nextScopes]);
      };
    };
    createScope.scopeName = baseScope.scopeName;
    return createScope;
  }

  // node_modules/@radix-ui/react-id/dist/index.mjs
  var React9 = __toESM(require_react(), 1);

  // node_modules/@radix-ui/react-use-layout-effect/dist/index.mjs
  var React8 = __toESM(require_react(), 1);
  var useLayoutEffect2 = globalThis?.document ? React8.useLayoutEffect : () => {
  };

  // node_modules/@radix-ui/react-id/dist/index.mjs
  var useReactId = React9[" useId ".trim().toString()] || (() => void 0);
  var count = 0;
  function useId(deterministicId) {
    const [id, setId] = React9.useState(useReactId());
    useLayoutEffect2(() => {
      if (!deterministicId) setId((reactId) => reactId ?? String(count++));
    }, [deterministicId]);
    return deterministicId || (id ? `radix-${id}` : "");
  }

  // node_modules/@radix-ui/react-use-controllable-state/dist/index.mjs
  var React10 = __toESM(require_react(), 1);
  var React22 = __toESM(require_react(), 1);
  var useInsertionEffect = React10[" useInsertionEffect ".trim().toString()] || useLayoutEffect2;
  function useControllableState({
    prop,
    defaultProp,
    onChange = () => {
    },
    caller
  }) {
    const [uncontrolledProp, setUncontrolledProp, onChangeRef] = useUncontrolledState({
      defaultProp,
      onChange
    });
    const isControlled = prop !== void 0;
    const value = isControlled ? prop : uncontrolledProp;
    if (true) {
      const isControlledRef = React10.useRef(prop !== void 0);
      React10.useEffect(() => {
        const wasControlled = isControlledRef.current;
        if (wasControlled !== isControlled) {
          const from = wasControlled ? "controlled" : "uncontrolled";
          const to = isControlled ? "controlled" : "uncontrolled";
          console.warn(
            `${caller} is changing from ${from} to ${to}. Components should not switch from controlled to uncontrolled (or vice versa). Decide between using a controlled or uncontrolled value for the lifetime of the component.`
          );
        }
        isControlledRef.current = isControlled;
      }, [isControlled, caller]);
    }
    const setValue = React10.useCallback(
      (nextValue) => {
        if (isControlled) {
          const value2 = isFunction(nextValue) ? nextValue(prop) : nextValue;
          if (value2 !== prop) {
            onChangeRef.current?.(value2);
          }
        } else {
          setUncontrolledProp(nextValue);
        }
      },
      [isControlled, prop, setUncontrolledProp, onChangeRef]
    );
    return [value, setValue];
  }
  function useUncontrolledState({
    defaultProp,
    onChange
  }) {
    const [value, setValue] = React10.useState(defaultProp);
    const prevValueRef = React10.useRef(value);
    const onChangeRef = React10.useRef(onChange);
    useInsertionEffect(() => {
      onChangeRef.current = onChange;
    }, [onChange]);
    React10.useEffect(() => {
      if (prevValueRef.current !== value) {
        onChangeRef.current?.(value);
        prevValueRef.current = value;
      }
    }, [value, prevValueRef]);
    return [value, setValue, onChangeRef];
  }
  function isFunction(value) {
    return typeof value === "function";
  }
  var SYNC_STATE = Symbol("RADIX:SYNC_STATE");

  // node_modules/@radix-ui/react-dismissable-layer/dist/index.mjs
  var React15 = __toESM(require_react(), 1);

  // node_modules/@radix-ui/react-primitive/dist/index.mjs
  var React12 = __toESM(require_react(), 1);
  var ReactDOM = __toESM(require_react_dom(), 1);

  // node_modules/@radix-ui/react-primitive/node_modules/@radix-ui/react-slot/dist/index.mjs
  var React11 = __toESM(require_react(), 1);
  var import_jsx_runtime7 = __toESM(require_jsx_runtime(), 1);
  // @__NO_SIDE_EFFECTS__
  function createSlot2(ownerName) {
    const SlotClone = /* @__PURE__ */ createSlotClone2(ownerName);
    const Slot22 = React11.forwardRef((props, forwardedRef) => {
      const { children, ...slotProps } = props;
      const childrenArray = React11.Children.toArray(children);
      const slottable = childrenArray.find(isSlottable2);
      if (slottable) {
        const newElement = slottable.props.children;
        const newChildren = childrenArray.map((child) => {
          if (child === slottable) {
            if (React11.Children.count(newElement) > 1) return React11.Children.only(null);
            return React11.isValidElement(newElement) ? newElement.props.children : null;
          } else {
            return child;
          }
        });
        return /* @__PURE__ */ (0, import_jsx_runtime7.jsx)(SlotClone, { ...slotProps, ref: forwardedRef, children: React11.isValidElement(newElement) ? React11.cloneElement(newElement, void 0, newChildren) : null });
      }
      return /* @__PURE__ */ (0, import_jsx_runtime7.jsx)(SlotClone, { ...slotProps, ref: forwardedRef, children });
    });
    Slot22.displayName = `${ownerName}.Slot`;
    return Slot22;
  }
  // @__NO_SIDE_EFFECTS__
  function createSlotClone2(ownerName) {
    const SlotClone = React11.forwardRef((props, forwardedRef) => {
      const { children, ...slotProps } = props;
      if (React11.isValidElement(children)) {
        const childrenRef = getElementRef2(children);
        const props2 = mergeProps2(slotProps, children.props);
        if (children.type !== React11.Fragment) {
          props2.ref = forwardedRef ? composeRefs(forwardedRef, childrenRef) : childrenRef;
        }
        return React11.cloneElement(children, props2);
      }
      return React11.Children.count(children) > 1 ? React11.Children.only(null) : null;
    });
    SlotClone.displayName = `${ownerName}.SlotClone`;
    return SlotClone;
  }
  var SLOTTABLE_IDENTIFIER2 = Symbol("radix.slottable");
  function isSlottable2(child) {
    return React11.isValidElement(child) && typeof child.type === "function" && "__radixId" in child.type && child.type.__radixId === SLOTTABLE_IDENTIFIER2;
  }
  function mergeProps2(slotProps, childProps) {
    const overrideProps = { ...childProps };
    for (const propName in childProps) {
      const slotPropValue = slotProps[propName];
      const childPropValue = childProps[propName];
      const isHandler = /^on[A-Z]/.test(propName);
      if (isHandler) {
        if (slotPropValue && childPropValue) {
          overrideProps[propName] = (...args) => {
            const result = childPropValue(...args);
            slotPropValue(...args);
            return result;
          };
        } else if (slotPropValue) {
          overrideProps[propName] = slotPropValue;
        }
      } else if (propName === "style") {
        overrideProps[propName] = { ...slotPropValue, ...childPropValue };
      } else if (propName === "className") {
        overrideProps[propName] = [slotPropValue, childPropValue].filter(Boolean).join(" ");
      }
    }
    return { ...slotProps, ...overrideProps };
  }
  function getElementRef2(element) {
    let getter = Object.getOwnPropertyDescriptor(element.props, "ref")?.get;
    let mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.ref;
    }
    getter = Object.getOwnPropertyDescriptor(element, "ref")?.get;
    mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.props.ref;
    }
    return element.props.ref || element.ref;
  }

  // node_modules/@radix-ui/react-primitive/dist/index.mjs
  var import_jsx_runtime8 = __toESM(require_jsx_runtime(), 1);
  var NODES = [
    "a",
    "button",
    "div",
    "form",
    "h2",
    "h3",
    "img",
    "input",
    "label",
    "li",
    "nav",
    "ol",
    "p",
    "select",
    "span",
    "svg",
    "ul"
  ];
  var Primitive = NODES.reduce((primitive, node) => {
    const Slot3 = createSlot2(`Primitive.${node}`);
    const Node2 = React12.forwardRef((props, forwardedRef) => {
      const { asChild, ...primitiveProps } = props;
      const Comp = asChild ? Slot3 : node;
      if (typeof window !== "undefined") {
        window[Symbol.for("radix-ui")] = true;
      }
      return /* @__PURE__ */ (0, import_jsx_runtime8.jsx)(Comp, { ...primitiveProps, ref: forwardedRef });
    });
    Node2.displayName = `Primitive.${node}`;
    return { ...primitive, [node]: Node2 };
  }, {});
  function dispatchDiscreteCustomEvent(target, event) {
    if (target) ReactDOM.flushSync(() => target.dispatchEvent(event));
  }

  // node_modules/@radix-ui/react-use-callback-ref/dist/index.mjs
  var React13 = __toESM(require_react(), 1);
  function useCallbackRef(callback) {
    const callbackRef = React13.useRef(callback);
    React13.useEffect(() => {
      callbackRef.current = callback;
    });
    return React13.useMemo(() => (...args) => callbackRef.current?.(...args), []);
  }

  // node_modules/@radix-ui/react-use-escape-keydown/dist/index.mjs
  var React14 = __toESM(require_react(), 1);
  function useEscapeKeydown(onEscapeKeyDownProp, ownerDocument = globalThis?.document) {
    const onEscapeKeyDown = useCallbackRef(onEscapeKeyDownProp);
    React14.useEffect(() => {
      const handleKeyDown = (event) => {
        if (event.key === "Escape") {
          onEscapeKeyDown(event);
        }
      };
      ownerDocument.addEventListener("keydown", handleKeyDown, { capture: true });
      return () => ownerDocument.removeEventListener("keydown", handleKeyDown, { capture: true });
    }, [onEscapeKeyDown, ownerDocument]);
  }

  // node_modules/@radix-ui/react-dismissable-layer/dist/index.mjs
  var import_jsx_runtime9 = __toESM(require_jsx_runtime(), 1);
  var DISMISSABLE_LAYER_NAME = "DismissableLayer";
  var CONTEXT_UPDATE = "dismissableLayer.update";
  var POINTER_DOWN_OUTSIDE = "dismissableLayer.pointerDownOutside";
  var FOCUS_OUTSIDE = "dismissableLayer.focusOutside";
  var originalBodyPointerEvents;
  var DismissableLayerContext = React15.createContext({
    layers: /* @__PURE__ */ new Set(),
    layersWithOutsidePointerEventsDisabled: /* @__PURE__ */ new Set(),
    branches: /* @__PURE__ */ new Set()
  });
  var DismissableLayer = React15.forwardRef(
    (props, forwardedRef) => {
      const {
        disableOutsidePointerEvents = false,
        onEscapeKeyDown,
        onPointerDownOutside,
        onFocusOutside,
        onInteractOutside,
        onDismiss,
        ...layerProps
      } = props;
      const context = React15.useContext(DismissableLayerContext);
      const [node, setNode] = React15.useState(null);
      const ownerDocument = node?.ownerDocument ?? globalThis?.document;
      const [, force] = React15.useState({});
      const composedRefs = useComposedRefs(forwardedRef, (node2) => setNode(node2));
      const layers = Array.from(context.layers);
      const [highestLayerWithOutsidePointerEventsDisabled] = [...context.layersWithOutsidePointerEventsDisabled].slice(-1);
      const highestLayerWithOutsidePointerEventsDisabledIndex = layers.indexOf(highestLayerWithOutsidePointerEventsDisabled);
      const index = node ? layers.indexOf(node) : -1;
      const isBodyPointerEventsDisabled = context.layersWithOutsidePointerEventsDisabled.size > 0;
      const isPointerEventsEnabled = index >= highestLayerWithOutsidePointerEventsDisabledIndex;
      const pointerDownOutside = usePointerDownOutside((event) => {
        const target = event.target;
        const isPointerDownOnBranch = [...context.branches].some((branch) => branch.contains(target));
        if (!isPointerEventsEnabled || isPointerDownOnBranch) return;
        onPointerDownOutside?.(event);
        onInteractOutside?.(event);
        if (!event.defaultPrevented) onDismiss?.();
      }, ownerDocument);
      const focusOutside = useFocusOutside((event) => {
        const target = event.target;
        const isFocusInBranch = [...context.branches].some((branch) => branch.contains(target));
        if (isFocusInBranch) return;
        onFocusOutside?.(event);
        onInteractOutside?.(event);
        if (!event.defaultPrevented) onDismiss?.();
      }, ownerDocument);
      useEscapeKeydown((event) => {
        const isHighestLayer = index === context.layers.size - 1;
        if (!isHighestLayer) return;
        onEscapeKeyDown?.(event);
        if (!event.defaultPrevented && onDismiss) {
          event.preventDefault();
          onDismiss();
        }
      }, ownerDocument);
      React15.useEffect(() => {
        if (!node) return;
        if (disableOutsidePointerEvents) {
          if (context.layersWithOutsidePointerEventsDisabled.size === 0) {
            originalBodyPointerEvents = ownerDocument.body.style.pointerEvents;
            ownerDocument.body.style.pointerEvents = "none";
          }
          context.layersWithOutsidePointerEventsDisabled.add(node);
        }
        context.layers.add(node);
        dispatchUpdate();
        return () => {
          if (disableOutsidePointerEvents && context.layersWithOutsidePointerEventsDisabled.size === 1) {
            ownerDocument.body.style.pointerEvents = originalBodyPointerEvents;
          }
        };
      }, [node, ownerDocument, disableOutsidePointerEvents, context]);
      React15.useEffect(() => {
        return () => {
          if (!node) return;
          context.layers.delete(node);
          context.layersWithOutsidePointerEventsDisabled.delete(node);
          dispatchUpdate();
        };
      }, [node, context]);
      React15.useEffect(() => {
        const handleUpdate = () => force({});
        document.addEventListener(CONTEXT_UPDATE, handleUpdate);
        return () => document.removeEventListener(CONTEXT_UPDATE, handleUpdate);
      }, []);
      return /* @__PURE__ */ (0, import_jsx_runtime9.jsx)(
        Primitive.div,
        {
          ...layerProps,
          ref: composedRefs,
          style: {
            pointerEvents: isBodyPointerEventsDisabled ? isPointerEventsEnabled ? "auto" : "none" : void 0,
            ...props.style
          },
          onFocusCapture: composeEventHandlers(props.onFocusCapture, focusOutside.onFocusCapture),
          onBlurCapture: composeEventHandlers(props.onBlurCapture, focusOutside.onBlurCapture),
          onPointerDownCapture: composeEventHandlers(
            props.onPointerDownCapture,
            pointerDownOutside.onPointerDownCapture
          )
        }
      );
    }
  );
  DismissableLayer.displayName = DISMISSABLE_LAYER_NAME;
  var BRANCH_NAME = "DismissableLayerBranch";
  var DismissableLayerBranch = React15.forwardRef((props, forwardedRef) => {
    const context = React15.useContext(DismissableLayerContext);
    const ref = React15.useRef(null);
    const composedRefs = useComposedRefs(forwardedRef, ref);
    React15.useEffect(() => {
      const node = ref.current;
      if (node) {
        context.branches.add(node);
        return () => {
          context.branches.delete(node);
        };
      }
    }, [context.branches]);
    return /* @__PURE__ */ (0, import_jsx_runtime9.jsx)(Primitive.div, { ...props, ref: composedRefs });
  });
  DismissableLayerBranch.displayName = BRANCH_NAME;
  function usePointerDownOutside(onPointerDownOutside, ownerDocument = globalThis?.document) {
    const handlePointerDownOutside = useCallbackRef(onPointerDownOutside);
    const isPointerInsideReactTreeRef = React15.useRef(false);
    const handleClickRef = React15.useRef(() => {
    });
    React15.useEffect(() => {
      const handlePointerDown = (event) => {
        if (event.target && !isPointerInsideReactTreeRef.current) {
          let handleAndDispatchPointerDownOutsideEvent2 = function() {
            handleAndDispatchCustomEvent(
              POINTER_DOWN_OUTSIDE,
              handlePointerDownOutside,
              eventDetail,
              { discrete: true }
            );
          };
          var handleAndDispatchPointerDownOutsideEvent = handleAndDispatchPointerDownOutsideEvent2;
          const eventDetail = { originalEvent: event };
          if (event.pointerType === "touch") {
            ownerDocument.removeEventListener("click", handleClickRef.current);
            handleClickRef.current = handleAndDispatchPointerDownOutsideEvent2;
            ownerDocument.addEventListener("click", handleClickRef.current, { once: true });
          } else {
            handleAndDispatchPointerDownOutsideEvent2();
          }
        } else {
          ownerDocument.removeEventListener("click", handleClickRef.current);
        }
        isPointerInsideReactTreeRef.current = false;
      };
      const timerId = window.setTimeout(() => {
        ownerDocument.addEventListener("pointerdown", handlePointerDown);
      }, 0);
      return () => {
        window.clearTimeout(timerId);
        ownerDocument.removeEventListener("pointerdown", handlePointerDown);
        ownerDocument.removeEventListener("click", handleClickRef.current);
      };
    }, [ownerDocument, handlePointerDownOutside]);
    return {
      // ensures we check React component tree (not just DOM tree)
      onPointerDownCapture: () => isPointerInsideReactTreeRef.current = true
    };
  }
  function useFocusOutside(onFocusOutside, ownerDocument = globalThis?.document) {
    const handleFocusOutside = useCallbackRef(onFocusOutside);
    const isFocusInsideReactTreeRef = React15.useRef(false);
    React15.useEffect(() => {
      const handleFocus = (event) => {
        if (event.target && !isFocusInsideReactTreeRef.current) {
          const eventDetail = { originalEvent: event };
          handleAndDispatchCustomEvent(FOCUS_OUTSIDE, handleFocusOutside, eventDetail, {
            discrete: false
          });
        }
      };
      ownerDocument.addEventListener("focusin", handleFocus);
      return () => ownerDocument.removeEventListener("focusin", handleFocus);
    }, [ownerDocument, handleFocusOutside]);
    return {
      onFocusCapture: () => isFocusInsideReactTreeRef.current = true,
      onBlurCapture: () => isFocusInsideReactTreeRef.current = false
    };
  }
  function dispatchUpdate() {
    const event = new CustomEvent(CONTEXT_UPDATE);
    document.dispatchEvent(event);
  }
  function handleAndDispatchCustomEvent(name, handler, detail, { discrete }) {
    const target = detail.originalEvent.target;
    const event = new CustomEvent(name, { bubbles: false, cancelable: true, detail });
    if (handler) target.addEventListener(name, handler, { once: true });
    if (discrete) {
      dispatchDiscreteCustomEvent(target, event);
    } else {
      target.dispatchEvent(event);
    }
  }

  // node_modules/@radix-ui/react-focus-scope/dist/index.mjs
  var React16 = __toESM(require_react(), 1);
  var import_jsx_runtime10 = __toESM(require_jsx_runtime(), 1);
  var AUTOFOCUS_ON_MOUNT = "focusScope.autoFocusOnMount";
  var AUTOFOCUS_ON_UNMOUNT = "focusScope.autoFocusOnUnmount";
  var EVENT_OPTIONS = { bubbles: false, cancelable: true };
  var FOCUS_SCOPE_NAME = "FocusScope";
  var FocusScope = React16.forwardRef((props, forwardedRef) => {
    const {
      loop = false,
      trapped = false,
      onMountAutoFocus: onMountAutoFocusProp,
      onUnmountAutoFocus: onUnmountAutoFocusProp,
      ...scopeProps
    } = props;
    const [container, setContainer] = React16.useState(null);
    const onMountAutoFocus = useCallbackRef(onMountAutoFocusProp);
    const onUnmountAutoFocus = useCallbackRef(onUnmountAutoFocusProp);
    const lastFocusedElementRef = React16.useRef(null);
    const composedRefs = useComposedRefs(forwardedRef, (node) => setContainer(node));
    const focusScope = React16.useRef({
      paused: false,
      pause() {
        this.paused = true;
      },
      resume() {
        this.paused = false;
      }
    }).current;
    React16.useEffect(() => {
      if (trapped) {
        let handleFocusIn2 = function(event) {
          if (focusScope.paused || !container) return;
          const target = event.target;
          if (container.contains(target)) {
            lastFocusedElementRef.current = target;
          } else {
            focus(lastFocusedElementRef.current, { select: true });
          }
        }, handleFocusOut2 = function(event) {
          if (focusScope.paused || !container) return;
          const relatedTarget = event.relatedTarget;
          if (relatedTarget === null) return;
          if (!container.contains(relatedTarget)) {
            focus(lastFocusedElementRef.current, { select: true });
          }
        }, handleMutations2 = function(mutations) {
          const focusedElement = document.activeElement;
          if (focusedElement !== document.body) return;
          for (const mutation of mutations) {
            if (mutation.removedNodes.length > 0) focus(container);
          }
        };
        var handleFocusIn = handleFocusIn2, handleFocusOut = handleFocusOut2, handleMutations = handleMutations2;
        document.addEventListener("focusin", handleFocusIn2);
        document.addEventListener("focusout", handleFocusOut2);
        const mutationObserver = new MutationObserver(handleMutations2);
        if (container) mutationObserver.observe(container, { childList: true, subtree: true });
        return () => {
          document.removeEventListener("focusin", handleFocusIn2);
          document.removeEventListener("focusout", handleFocusOut2);
          mutationObserver.disconnect();
        };
      }
    }, [trapped, container, focusScope.paused]);
    React16.useEffect(() => {
      if (container) {
        focusScopesStack.add(focusScope);
        const previouslyFocusedElement = document.activeElement;
        const hasFocusedCandidate = container.contains(previouslyFocusedElement);
        if (!hasFocusedCandidate) {
          const mountEvent = new CustomEvent(AUTOFOCUS_ON_MOUNT, EVENT_OPTIONS);
          container.addEventListener(AUTOFOCUS_ON_MOUNT, onMountAutoFocus);
          container.dispatchEvent(mountEvent);
          if (!mountEvent.defaultPrevented) {
            focusFirst(removeLinks(getTabbableCandidates(container)), { select: true });
            if (document.activeElement === previouslyFocusedElement) {
              focus(container);
            }
          }
        }
        return () => {
          container.removeEventListener(AUTOFOCUS_ON_MOUNT, onMountAutoFocus);
          setTimeout(() => {
            const unmountEvent = new CustomEvent(AUTOFOCUS_ON_UNMOUNT, EVENT_OPTIONS);
            container.addEventListener(AUTOFOCUS_ON_UNMOUNT, onUnmountAutoFocus);
            container.dispatchEvent(unmountEvent);
            if (!unmountEvent.defaultPrevented) {
              focus(previouslyFocusedElement ?? document.body, { select: true });
            }
            container.removeEventListener(AUTOFOCUS_ON_UNMOUNT, onUnmountAutoFocus);
            focusScopesStack.remove(focusScope);
          }, 0);
        };
      }
    }, [container, onMountAutoFocus, onUnmountAutoFocus, focusScope]);
    const handleKeyDown = React16.useCallback(
      (event) => {
        if (!loop && !trapped) return;
        if (focusScope.paused) return;
        const isTabKey = event.key === "Tab" && !event.altKey && !event.ctrlKey && !event.metaKey;
        const focusedElement = document.activeElement;
        if (isTabKey && focusedElement) {
          const container2 = event.currentTarget;
          const [first, last] = getTabbableEdges(container2);
          const hasTabbableElementsInside = first && last;
          if (!hasTabbableElementsInside) {
            if (focusedElement === container2) event.preventDefault();
          } else {
            if (!event.shiftKey && focusedElement === last) {
              event.preventDefault();
              if (loop) focus(first, { select: true });
            } else if (event.shiftKey && focusedElement === first) {
              event.preventDefault();
              if (loop) focus(last, { select: true });
            }
          }
        }
      },
      [loop, trapped, focusScope.paused]
    );
    return /* @__PURE__ */ (0, import_jsx_runtime10.jsx)(Primitive.div, { tabIndex: -1, ...scopeProps, ref: composedRefs, onKeyDown: handleKeyDown });
  });
  FocusScope.displayName = FOCUS_SCOPE_NAME;
  function focusFirst(candidates, { select = false } = {}) {
    const previouslyFocusedElement = document.activeElement;
    for (const candidate of candidates) {
      focus(candidate, { select });
      if (document.activeElement !== previouslyFocusedElement) return;
    }
  }
  function getTabbableEdges(container) {
    const candidates = getTabbableCandidates(container);
    const first = findVisible(candidates, container);
    const last = findVisible(candidates.reverse(), container);
    return [first, last];
  }
  function getTabbableCandidates(container) {
    const nodes = [];
    const walker = document.createTreeWalker(container, NodeFilter.SHOW_ELEMENT, {
      acceptNode: (node) => {
        const isHiddenInput = node.tagName === "INPUT" && node.type === "hidden";
        if (node.disabled || node.hidden || isHiddenInput) return NodeFilter.FILTER_SKIP;
        return node.tabIndex >= 0 ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP;
      }
    });
    while (walker.nextNode()) nodes.push(walker.currentNode);
    return nodes;
  }
  function findVisible(elements, container) {
    for (const element of elements) {
      if (!isHidden(element, { upTo: container })) return element;
    }
  }
  function isHidden(node, { upTo }) {
    if (getComputedStyle(node).visibility === "hidden") return true;
    while (node) {
      if (upTo !== void 0 && node === upTo) return false;
      if (getComputedStyle(node).display === "none") return true;
      node = node.parentElement;
    }
    return false;
  }
  function isSelectableInput(element) {
    return element instanceof HTMLInputElement && "select" in element;
  }
  function focus(element, { select = false } = {}) {
    if (element && element.focus) {
      const previouslyFocusedElement = document.activeElement;
      element.focus({ preventScroll: true });
      if (element !== previouslyFocusedElement && isSelectableInput(element) && select)
        element.select();
    }
  }
  var focusScopesStack = createFocusScopesStack();
  function createFocusScopesStack() {
    let stack = [];
    return {
      add(focusScope) {
        const activeFocusScope = stack[0];
        if (focusScope !== activeFocusScope) {
          activeFocusScope?.pause();
        }
        stack = arrayRemove(stack, focusScope);
        stack.unshift(focusScope);
      },
      remove(focusScope) {
        stack = arrayRemove(stack, focusScope);
        stack[0]?.resume();
      }
    };
  }
  function arrayRemove(array, item) {
    const updatedArray = [...array];
    const index = updatedArray.indexOf(item);
    if (index !== -1) {
      updatedArray.splice(index, 1);
    }
    return updatedArray;
  }
  function removeLinks(items) {
    return items.filter((item) => item.tagName !== "A");
  }

  // node_modules/@radix-ui/react-portal/dist/index.mjs
  var React17 = __toESM(require_react(), 1);
  var import_react_dom = __toESM(require_react_dom(), 1);
  var import_jsx_runtime11 = __toESM(require_jsx_runtime(), 1);
  var PORTAL_NAME = "Portal";
  var Portal = React17.forwardRef((props, forwardedRef) => {
    const { container: containerProp, ...portalProps } = props;
    const [mounted, setMounted] = React17.useState(false);
    useLayoutEffect2(() => setMounted(true), []);
    const container = containerProp || mounted && globalThis?.document?.body;
    return container ? import_react_dom.default.createPortal(/* @__PURE__ */ (0, import_jsx_runtime11.jsx)(Primitive.div, { ...portalProps, ref: forwardedRef }), container) : null;
  });
  Portal.displayName = PORTAL_NAME;

  // node_modules/@radix-ui/react-presence/dist/index.mjs
  var React23 = __toESM(require_react(), 1);
  var React18 = __toESM(require_react(), 1);
  function useStateMachine(initialState, machine) {
    return React18.useReducer((state, event) => {
      const nextState = machine[state][event];
      return nextState ?? state;
    }, initialState);
  }
  var Presence = (props) => {
    const { present, children } = props;
    const presence = usePresence(present);
    const child = typeof children === "function" ? children({ present: presence.isPresent }) : React23.Children.only(children);
    const ref = useComposedRefs(presence.ref, getElementRef3(child));
    const forceMount = typeof children === "function";
    return forceMount || presence.isPresent ? React23.cloneElement(child, { ref }) : null;
  };
  Presence.displayName = "Presence";
  function usePresence(present) {
    const [node, setNode] = React23.useState();
    const stylesRef = React23.useRef(null);
    const prevPresentRef = React23.useRef(present);
    const prevAnimationNameRef = React23.useRef("none");
    const initialState = present ? "mounted" : "unmounted";
    const [state, send] = useStateMachine(initialState, {
      mounted: {
        UNMOUNT: "unmounted",
        ANIMATION_OUT: "unmountSuspended"
      },
      unmountSuspended: {
        MOUNT: "mounted",
        ANIMATION_END: "unmounted"
      },
      unmounted: {
        MOUNT: "mounted"
      }
    });
    React23.useEffect(() => {
      const currentAnimationName = getAnimationName(stylesRef.current);
      prevAnimationNameRef.current = state === "mounted" ? currentAnimationName : "none";
    }, [state]);
    useLayoutEffect2(() => {
      const styles = stylesRef.current;
      const wasPresent = prevPresentRef.current;
      const hasPresentChanged = wasPresent !== present;
      if (hasPresentChanged) {
        const prevAnimationName = prevAnimationNameRef.current;
        const currentAnimationName = getAnimationName(styles);
        if (present) {
          send("MOUNT");
        } else if (currentAnimationName === "none" || styles?.display === "none") {
          send("UNMOUNT");
        } else {
          const isAnimating = prevAnimationName !== currentAnimationName;
          if (wasPresent && isAnimating) {
            send("ANIMATION_OUT");
          } else {
            send("UNMOUNT");
          }
        }
        prevPresentRef.current = present;
      }
    }, [present, send]);
    useLayoutEffect2(() => {
      if (node) {
        let timeoutId;
        const ownerWindow = node.ownerDocument.defaultView ?? window;
        const handleAnimationEnd = (event) => {
          const currentAnimationName = getAnimationName(stylesRef.current);
          const isCurrentAnimation = currentAnimationName.includes(CSS.escape(event.animationName));
          if (event.target === node && isCurrentAnimation) {
            send("ANIMATION_END");
            if (!prevPresentRef.current) {
              const currentFillMode = node.style.animationFillMode;
              node.style.animationFillMode = "forwards";
              timeoutId = ownerWindow.setTimeout(() => {
                if (node.style.animationFillMode === "forwards") {
                  node.style.animationFillMode = currentFillMode;
                }
              });
            }
          }
        };
        const handleAnimationStart = (event) => {
          if (event.target === node) {
            prevAnimationNameRef.current = getAnimationName(stylesRef.current);
          }
        };
        node.addEventListener("animationstart", handleAnimationStart);
        node.addEventListener("animationcancel", handleAnimationEnd);
        node.addEventListener("animationend", handleAnimationEnd);
        return () => {
          ownerWindow.clearTimeout(timeoutId);
          node.removeEventListener("animationstart", handleAnimationStart);
          node.removeEventListener("animationcancel", handleAnimationEnd);
          node.removeEventListener("animationend", handleAnimationEnd);
        };
      } else {
        send("ANIMATION_END");
      }
    }, [node, send]);
    return {
      isPresent: ["mounted", "unmountSuspended"].includes(state),
      ref: React23.useCallback((node2) => {
        stylesRef.current = node2 ? getComputedStyle(node2) : null;
        setNode(node2);
      }, [])
    };
  }
  function getAnimationName(styles) {
    return styles?.animationName || "none";
  }
  function getElementRef3(element) {
    let getter = Object.getOwnPropertyDescriptor(element.props, "ref")?.get;
    let mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.ref;
    }
    getter = Object.getOwnPropertyDescriptor(element, "ref")?.get;
    mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.props.ref;
    }
    return element.props.ref || element.ref;
  }

  // node_modules/@radix-ui/react-focus-guards/dist/index.mjs
  var React19 = __toESM(require_react(), 1);
  var count2 = 0;
  function useFocusGuards() {
    React19.useEffect(() => {
      const edgeGuards = document.querySelectorAll("[data-radix-focus-guard]");
      document.body.insertAdjacentElement("afterbegin", edgeGuards[0] ?? createFocusGuard());
      document.body.insertAdjacentElement("beforeend", edgeGuards[1] ?? createFocusGuard());
      count2++;
      return () => {
        if (count2 === 1) {
          document.querySelectorAll("[data-radix-focus-guard]").forEach((node) => node.remove());
        }
        count2--;
      };
    }, []);
  }
  function createFocusGuard() {
    const element = document.createElement("span");
    element.setAttribute("data-radix-focus-guard", "");
    element.tabIndex = 0;
    element.style.outline = "none";
    element.style.opacity = "0";
    element.style.position = "fixed";
    element.style.pointerEvents = "none";
    return element;
  }

  // node_modules/tslib/tslib.es6.mjs
  var __assign = function() {
    __assign = Object.assign || function __assign2(t) {
      for (var s, i = 1, n = arguments.length; i < n; i++) {
        s = arguments[i];
        for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p)) t[p] = s[p];
      }
      return t;
    };
    return __assign.apply(this, arguments);
  };
  function __rest(s, e) {
    var t = {};
    for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p) && e.indexOf(p) < 0)
      t[p] = s[p];
    if (s != null && typeof Object.getOwnPropertySymbols === "function")
      for (var i = 0, p = Object.getOwnPropertySymbols(s); i < p.length; i++) {
        if (e.indexOf(p[i]) < 0 && Object.prototype.propertyIsEnumerable.call(s, p[i]))
          t[p[i]] = s[p[i]];
      }
    return t;
  }
  function __spreadArray(to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
      if (ar || !(i in from)) {
        if (!ar) ar = Array.prototype.slice.call(from, 0, i);
        ar[i] = from[i];
      }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
  }

  // node_modules/react-remove-scroll/dist/es2015/Combination.js
  var React28 = __toESM(require_react());

  // node_modules/react-remove-scroll/dist/es2015/UI.js
  var React24 = __toESM(require_react());

  // node_modules/react-remove-scroll-bar/dist/es2015/constants.js
  var zeroRightClassName = "right-scroll-bar-position";
  var fullWidthClassName = "width-before-scroll-bar";
  var noScrollbarsClassName = "with-scroll-bars-hidden";
  var removedBarSizeVariable = "--removed-body-scroll-bar-size";

  // node_modules/use-callback-ref/dist/es2015/assignRef.js
  function assignRef(ref, value) {
    if (typeof ref === "function") {
      ref(value);
    } else if (ref) {
      ref.current = value;
    }
    return ref;
  }

  // node_modules/use-callback-ref/dist/es2015/useRef.js
  var import_react5 = __toESM(require_react());
  function useCallbackRef2(initialValue, callback) {
    var ref = (0, import_react5.useState)(function() {
      return {
        // value
        value: initialValue,
        // last callback
        callback,
        // "memoized" public interface
        facade: {
          get current() {
            return ref.value;
          },
          set current(value) {
            var last = ref.value;
            if (last !== value) {
              ref.value = value;
              ref.callback(value, last);
            }
          }
        }
      };
    })[0];
    ref.callback = callback;
    return ref.facade;
  }

  // node_modules/use-callback-ref/dist/es2015/useMergeRef.js
  var React20 = __toESM(require_react());
  var useIsomorphicLayoutEffect = typeof window !== "undefined" ? React20.useLayoutEffect : React20.useEffect;
  var currentValues = /* @__PURE__ */ new WeakMap();
  function useMergeRefs(refs, defaultValue) {
    var callbackRef = useCallbackRef2(defaultValue || null, function(newValue) {
      return refs.forEach(function(ref) {
        return assignRef(ref, newValue);
      });
    });
    useIsomorphicLayoutEffect(function() {
      var oldValue = currentValues.get(callbackRef);
      if (oldValue) {
        var prevRefs_1 = new Set(oldValue);
        var nextRefs_1 = new Set(refs);
        var current_1 = callbackRef.current;
        prevRefs_1.forEach(function(ref) {
          if (!nextRefs_1.has(ref)) {
            assignRef(ref, null);
          }
        });
        nextRefs_1.forEach(function(ref) {
          if (!prevRefs_1.has(ref)) {
            assignRef(ref, current_1);
          }
        });
      }
      currentValues.set(callbackRef, refs);
    }, [refs]);
    return callbackRef;
  }

  // node_modules/use-sidecar/dist/es2015/medium.js
  function ItoI(a) {
    return a;
  }
  function innerCreateMedium(defaults, middleware) {
    if (middleware === void 0) {
      middleware = ItoI;
    }
    var buffer = [];
    var assigned = false;
    var medium = {
      read: function() {
        if (assigned) {
          throw new Error("Sidecar: could not `read` from an `assigned` medium. `read` could be used only with `useMedium`.");
        }
        if (buffer.length) {
          return buffer[buffer.length - 1];
        }
        return defaults;
      },
      useMedium: function(data) {
        var item = middleware(data, assigned);
        buffer.push(item);
        return function() {
          buffer = buffer.filter(function(x) {
            return x !== item;
          });
        };
      },
      assignSyncMedium: function(cb) {
        assigned = true;
        while (buffer.length) {
          var cbs = buffer;
          buffer = [];
          cbs.forEach(cb);
        }
        buffer = {
          push: function(x) {
            return cb(x);
          },
          filter: function() {
            return buffer;
          }
        };
      },
      assignMedium: function(cb) {
        assigned = true;
        var pendingQueue = [];
        if (buffer.length) {
          var cbs = buffer;
          buffer = [];
          cbs.forEach(cb);
          pendingQueue = buffer;
        }
        var executeQueue = function() {
          var cbs2 = pendingQueue;
          pendingQueue = [];
          cbs2.forEach(cb);
        };
        var cycle = function() {
          return Promise.resolve().then(executeQueue);
        };
        cycle();
        buffer = {
          push: function(x) {
            pendingQueue.push(x);
            cycle();
          },
          filter: function(filter) {
            pendingQueue = pendingQueue.filter(filter);
            return buffer;
          }
        };
      }
    };
    return medium;
  }
  function createSidecarMedium(options) {
    if (options === void 0) {
      options = {};
    }
    var medium = innerCreateMedium(null);
    medium.options = __assign({ async: true, ssr: false }, options);
    return medium;
  }

  // node_modules/use-sidecar/dist/es2015/exports.js
  var React21 = __toESM(require_react());
  var SideCar = function(_a) {
    var sideCar = _a.sideCar, rest = __rest(_a, ["sideCar"]);
    if (!sideCar) {
      throw new Error("Sidecar: please provide `sideCar` property to import the right car");
    }
    var Target = sideCar.read();
    if (!Target) {
      throw new Error("Sidecar medium not found");
    }
    return React21.createElement(Target, __assign({}, rest));
  };
  SideCar.isSideCarExport = true;
  function exportSidecar(medium, exported) {
    medium.useMedium(exported);
    return SideCar;
  }

  // node_modules/react-remove-scroll/dist/es2015/medium.js
  var effectCar = createSidecarMedium();

  // node_modules/react-remove-scroll/dist/es2015/UI.js
  var nothing = function() {
    return;
  };
  var RemoveScroll = React24.forwardRef(function(props, parentRef) {
    var ref = React24.useRef(null);
    var _a = React24.useState({
      onScrollCapture: nothing,
      onWheelCapture: nothing,
      onTouchMoveCapture: nothing
    }), callbacks = _a[0], setCallbacks = _a[1];
    var forwardProps = props.forwardProps, children = props.children, className = props.className, removeScrollBar = props.removeScrollBar, enabled = props.enabled, shards = props.shards, sideCar = props.sideCar, noRelative = props.noRelative, noIsolation = props.noIsolation, inert = props.inert, allowPinchZoom = props.allowPinchZoom, _b = props.as, Container = _b === void 0 ? "div" : _b, gapMode = props.gapMode, rest = __rest(props, ["forwardProps", "children", "className", "removeScrollBar", "enabled", "shards", "sideCar", "noRelative", "noIsolation", "inert", "allowPinchZoom", "as", "gapMode"]);
    var SideCar2 = sideCar;
    var containerRef = useMergeRefs([ref, parentRef]);
    var containerProps = __assign(__assign({}, rest), callbacks);
    return React24.createElement(
      React24.Fragment,
      null,
      enabled && React24.createElement(SideCar2, { sideCar: effectCar, removeScrollBar, shards, noRelative, noIsolation, inert, setCallbacks, allowPinchZoom: !!allowPinchZoom, lockRef: ref, gapMode }),
      forwardProps ? React24.cloneElement(React24.Children.only(children), __assign(__assign({}, containerProps), { ref: containerRef })) : React24.createElement(Container, __assign({}, containerProps, { className, ref: containerRef }), children)
    );
  });
  RemoveScroll.defaultProps = {
    enabled: true,
    removeScrollBar: true,
    inert: false
  };
  RemoveScroll.classNames = {
    fullWidth: fullWidthClassName,
    zeroRight: zeroRightClassName
  };

  // node_modules/react-remove-scroll/dist/es2015/SideEffect.js
  var React27 = __toESM(require_react());

  // node_modules/react-remove-scroll-bar/dist/es2015/component.js
  var React26 = __toESM(require_react());

  // node_modules/react-style-singleton/dist/es2015/hook.js
  var React25 = __toESM(require_react());

  // node_modules/get-nonce/dist/es2015/index.js
  var currentNonce;
  var getNonce = function() {
    if (currentNonce) {
      return currentNonce;
    }
    if (typeof __webpack_nonce__ !== "undefined") {
      return __webpack_nonce__;
    }
    return void 0;
  };

  // node_modules/react-style-singleton/dist/es2015/singleton.js
  function makeStyleTag() {
    if (!document)
      return null;
    var tag = document.createElement("style");
    tag.type = "text/css";
    var nonce = getNonce();
    if (nonce) {
      tag.setAttribute("nonce", nonce);
    }
    return tag;
  }
  function injectStyles(tag, css) {
    if (tag.styleSheet) {
      tag.styleSheet.cssText = css;
    } else {
      tag.appendChild(document.createTextNode(css));
    }
  }
  function insertStyleTag(tag) {
    var head = document.head || document.getElementsByTagName("head")[0];
    head.appendChild(tag);
  }
  var stylesheetSingleton = function() {
    var counter = 0;
    var stylesheet = null;
    return {
      add: function(style) {
        if (counter == 0) {
          if (stylesheet = makeStyleTag()) {
            injectStyles(stylesheet, style);
            insertStyleTag(stylesheet);
          }
        }
        counter++;
      },
      remove: function() {
        counter--;
        if (!counter && stylesheet) {
          stylesheet.parentNode && stylesheet.parentNode.removeChild(stylesheet);
          stylesheet = null;
        }
      }
    };
  };

  // node_modules/react-style-singleton/dist/es2015/hook.js
  var styleHookSingleton = function() {
    var sheet = stylesheetSingleton();
    return function(styles, isDynamic) {
      React25.useEffect(function() {
        sheet.add(styles);
        return function() {
          sheet.remove();
        };
      }, [styles && isDynamic]);
    };
  };

  // node_modules/react-style-singleton/dist/es2015/component.js
  var styleSingleton = function() {
    var useStyle = styleHookSingleton();
    var Sheet = function(_a) {
      var styles = _a.styles, dynamic = _a.dynamic;
      useStyle(styles, dynamic);
      return null;
    };
    return Sheet;
  };

  // node_modules/react-remove-scroll-bar/dist/es2015/utils.js
  var zeroGap = {
    left: 0,
    top: 0,
    right: 0,
    gap: 0
  };
  var parse = function(x) {
    return parseInt(x || "", 10) || 0;
  };
  var getOffset = function(gapMode) {
    var cs = window.getComputedStyle(document.body);
    var left = cs[gapMode === "padding" ? "paddingLeft" : "marginLeft"];
    var top = cs[gapMode === "padding" ? "paddingTop" : "marginTop"];
    var right = cs[gapMode === "padding" ? "paddingRight" : "marginRight"];
    return [parse(left), parse(top), parse(right)];
  };
  var getGapWidth = function(gapMode) {
    if (gapMode === void 0) {
      gapMode = "margin";
    }
    if (typeof window === "undefined") {
      return zeroGap;
    }
    var offsets = getOffset(gapMode);
    var documentWidth = document.documentElement.clientWidth;
    var windowWidth = window.innerWidth;
    return {
      left: offsets[0],
      top: offsets[1],
      right: offsets[2],
      gap: Math.max(0, windowWidth - documentWidth + offsets[2] - offsets[0])
    };
  };

  // node_modules/react-remove-scroll-bar/dist/es2015/component.js
  var Style = styleSingleton();
  var lockAttribute = "data-scroll-locked";
  var getStyles = function(_a, allowRelative, gapMode, important) {
    var left = _a.left, top = _a.top, right = _a.right, gap = _a.gap;
    if (gapMode === void 0) {
      gapMode = "margin";
    }
    return "\n  .".concat(noScrollbarsClassName, " {\n   overflow: hidden ").concat(important, ";\n   padding-right: ").concat(gap, "px ").concat(important, ";\n  }\n  body[").concat(lockAttribute, "] {\n    overflow: hidden ").concat(important, ";\n    overscroll-behavior: contain;\n    ").concat([
      allowRelative && "position: relative ".concat(important, ";"),
      gapMode === "margin" && "\n    padding-left: ".concat(left, "px;\n    padding-top: ").concat(top, "px;\n    padding-right: ").concat(right, "px;\n    margin-left:0;\n    margin-top:0;\n    margin-right: ").concat(gap, "px ").concat(important, ";\n    "),
      gapMode === "padding" && "padding-right: ".concat(gap, "px ").concat(important, ";")
    ].filter(Boolean).join(""), "\n  }\n  \n  .").concat(zeroRightClassName, " {\n    right: ").concat(gap, "px ").concat(important, ";\n  }\n  \n  .").concat(fullWidthClassName, " {\n    margin-right: ").concat(gap, "px ").concat(important, ";\n  }\n  \n  .").concat(zeroRightClassName, " .").concat(zeroRightClassName, " {\n    right: 0 ").concat(important, ";\n  }\n  \n  .").concat(fullWidthClassName, " .").concat(fullWidthClassName, " {\n    margin-right: 0 ").concat(important, ";\n  }\n  \n  body[").concat(lockAttribute, "] {\n    ").concat(removedBarSizeVariable, ": ").concat(gap, "px;\n  }\n");
  };
  var getCurrentUseCounter = function() {
    var counter = parseInt(document.body.getAttribute(lockAttribute) || "0", 10);
    return isFinite(counter) ? counter : 0;
  };
  var useLockAttribute = function() {
    React26.useEffect(function() {
      document.body.setAttribute(lockAttribute, (getCurrentUseCounter() + 1).toString());
      return function() {
        var newCounter = getCurrentUseCounter() - 1;
        if (newCounter <= 0) {
          document.body.removeAttribute(lockAttribute);
        } else {
          document.body.setAttribute(lockAttribute, newCounter.toString());
        }
      };
    }, []);
  };
  var RemoveScrollBar = function(_a) {
    var noRelative = _a.noRelative, noImportant = _a.noImportant, _b = _a.gapMode, gapMode = _b === void 0 ? "margin" : _b;
    useLockAttribute();
    var gap = React26.useMemo(function() {
      return getGapWidth(gapMode);
    }, [gapMode]);
    return React26.createElement(Style, { styles: getStyles(gap, !noRelative, gapMode, !noImportant ? "!important" : "") });
  };

  // node_modules/react-remove-scroll/dist/es2015/aggresiveCapture.js
  var passiveSupported = false;
  if (typeof window !== "undefined") {
    try {
      options = Object.defineProperty({}, "passive", {
        get: function() {
          passiveSupported = true;
          return true;
        }
      });
      window.addEventListener("test", options, options);
      window.removeEventListener("test", options, options);
    } catch (err) {
      passiveSupported = false;
    }
  }
  var options;
  var nonPassive = passiveSupported ? { passive: false } : false;

  // node_modules/react-remove-scroll/dist/es2015/handleScroll.js
  var alwaysContainsScroll = function(node) {
    return node.tagName === "TEXTAREA";
  };
  var elementCanBeScrolled = function(node, overflow) {
    if (!(node instanceof Element)) {
      return false;
    }
    var styles = window.getComputedStyle(node);
    return (
      // not-not-scrollable
      styles[overflow] !== "hidden" && // contains scroll inside self
      !(styles.overflowY === styles.overflowX && !alwaysContainsScroll(node) && styles[overflow] === "visible")
    );
  };
  var elementCouldBeVScrolled = function(node) {
    return elementCanBeScrolled(node, "overflowY");
  };
  var elementCouldBeHScrolled = function(node) {
    return elementCanBeScrolled(node, "overflowX");
  };
  var locationCouldBeScrolled = function(axis, node) {
    var ownerDocument = node.ownerDocument;
    var current = node;
    do {
      if (typeof ShadowRoot !== "undefined" && current instanceof ShadowRoot) {
        current = current.host;
      }
      var isScrollable = elementCouldBeScrolled(axis, current);
      if (isScrollable) {
        var _a = getScrollVariables(axis, current), scrollHeight = _a[1], clientHeight = _a[2];
        if (scrollHeight > clientHeight) {
          return true;
        }
      }
      current = current.parentNode;
    } while (current && current !== ownerDocument.body);
    return false;
  };
  var getVScrollVariables = function(_a) {
    var scrollTop = _a.scrollTop, scrollHeight = _a.scrollHeight, clientHeight = _a.clientHeight;
    return [
      scrollTop,
      scrollHeight,
      clientHeight
    ];
  };
  var getHScrollVariables = function(_a) {
    var scrollLeft = _a.scrollLeft, scrollWidth = _a.scrollWidth, clientWidth = _a.clientWidth;
    return [
      scrollLeft,
      scrollWidth,
      clientWidth
    ];
  };
  var elementCouldBeScrolled = function(axis, node) {
    return axis === "v" ? elementCouldBeVScrolled(node) : elementCouldBeHScrolled(node);
  };
  var getScrollVariables = function(axis, node) {
    return axis === "v" ? getVScrollVariables(node) : getHScrollVariables(node);
  };
  var getDirectionFactor = function(axis, direction) {
    return axis === "h" && direction === "rtl" ? -1 : 1;
  };
  var handleScroll = function(axis, endTarget, event, sourceDelta, noOverscroll) {
    var directionFactor = getDirectionFactor(axis, window.getComputedStyle(endTarget).direction);
    var delta = directionFactor * sourceDelta;
    var target = event.target;
    var targetInLock = endTarget.contains(target);
    var shouldCancelScroll = false;
    var isDeltaPositive = delta > 0;
    var availableScroll = 0;
    var availableScrollTop = 0;
    do {
      if (!target) {
        break;
      }
      var _a = getScrollVariables(axis, target), position = _a[0], scroll_1 = _a[1], capacity = _a[2];
      var elementScroll = scroll_1 - capacity - directionFactor * position;
      if (position || elementScroll) {
        if (elementCouldBeScrolled(axis, target)) {
          availableScroll += elementScroll;
          availableScrollTop += position;
        }
      }
      var parent_1 = target.parentNode;
      target = parent_1 && parent_1.nodeType === Node.DOCUMENT_FRAGMENT_NODE ? parent_1.host : parent_1;
    } while (
      // portaled content
      !targetInLock && target !== document.body || // self content
      targetInLock && (endTarget.contains(target) || endTarget === target)
    );
    if (isDeltaPositive && (noOverscroll && Math.abs(availableScroll) < 1 || !noOverscroll && delta > availableScroll)) {
      shouldCancelScroll = true;
    } else if (!isDeltaPositive && (noOverscroll && Math.abs(availableScrollTop) < 1 || !noOverscroll && -delta > availableScrollTop)) {
      shouldCancelScroll = true;
    }
    return shouldCancelScroll;
  };

  // node_modules/react-remove-scroll/dist/es2015/SideEffect.js
  var getTouchXY = function(event) {
    return "changedTouches" in event ? [event.changedTouches[0].clientX, event.changedTouches[0].clientY] : [0, 0];
  };
  var getDeltaXY = function(event) {
    return [event.deltaX, event.deltaY];
  };
  var extractRef = function(ref) {
    return ref && "current" in ref ? ref.current : ref;
  };
  var deltaCompare = function(x, y) {
    return x[0] === y[0] && x[1] === y[1];
  };
  var generateStyle = function(id) {
    return "\n  .block-interactivity-".concat(id, " {pointer-events: none;}\n  .allow-interactivity-").concat(id, " {pointer-events: all;}\n");
  };
  var idCounter = 0;
  var lockStack = [];
  function RemoveScrollSideCar(props) {
    var shouldPreventQueue = React27.useRef([]);
    var touchStartRef = React27.useRef([0, 0]);
    var activeAxis = React27.useRef();
    var id = React27.useState(idCounter++)[0];
    var Style2 = React27.useState(styleSingleton)[0];
    var lastProps = React27.useRef(props);
    React27.useEffect(function() {
      lastProps.current = props;
    }, [props]);
    React27.useEffect(function() {
      if (props.inert) {
        document.body.classList.add("block-interactivity-".concat(id));
        var allow_1 = __spreadArray([props.lockRef.current], (props.shards || []).map(extractRef), true).filter(Boolean);
        allow_1.forEach(function(el) {
          return el.classList.add("allow-interactivity-".concat(id));
        });
        return function() {
          document.body.classList.remove("block-interactivity-".concat(id));
          allow_1.forEach(function(el) {
            return el.classList.remove("allow-interactivity-".concat(id));
          });
        };
      }
      return;
    }, [props.inert, props.lockRef.current, props.shards]);
    var shouldCancelEvent = React27.useCallback(function(event, parent) {
      if ("touches" in event && event.touches.length === 2 || event.type === "wheel" && event.ctrlKey) {
        return !lastProps.current.allowPinchZoom;
      }
      var touch = getTouchXY(event);
      var touchStart = touchStartRef.current;
      var deltaX = "deltaX" in event ? event.deltaX : touchStart[0] - touch[0];
      var deltaY = "deltaY" in event ? event.deltaY : touchStart[1] - touch[1];
      var currentAxis;
      var target = event.target;
      var moveDirection = Math.abs(deltaX) > Math.abs(deltaY) ? "h" : "v";
      if ("touches" in event && moveDirection === "h" && target.type === "range") {
        return false;
      }
      var selection = window.getSelection();
      var anchorNode = selection && selection.anchorNode;
      var isTouchingSelection = anchorNode ? anchorNode === target || anchorNode.contains(target) : false;
      if (isTouchingSelection) {
        return false;
      }
      var canBeScrolledInMainDirection = locationCouldBeScrolled(moveDirection, target);
      if (!canBeScrolledInMainDirection) {
        return true;
      }
      if (canBeScrolledInMainDirection) {
        currentAxis = moveDirection;
      } else {
        currentAxis = moveDirection === "v" ? "h" : "v";
        canBeScrolledInMainDirection = locationCouldBeScrolled(moveDirection, target);
      }
      if (!canBeScrolledInMainDirection) {
        return false;
      }
      if (!activeAxis.current && "changedTouches" in event && (deltaX || deltaY)) {
        activeAxis.current = currentAxis;
      }
      if (!currentAxis) {
        return true;
      }
      var cancelingAxis = activeAxis.current || currentAxis;
      return handleScroll(cancelingAxis, parent, event, cancelingAxis === "h" ? deltaX : deltaY, true);
    }, []);
    var shouldPrevent = React27.useCallback(function(_event) {
      var event = _event;
      if (!lockStack.length || lockStack[lockStack.length - 1] !== Style2) {
        return;
      }
      var delta = "deltaY" in event ? getDeltaXY(event) : getTouchXY(event);
      var sourceEvent = shouldPreventQueue.current.filter(function(e) {
        return e.name === event.type && (e.target === event.target || event.target === e.shadowParent) && deltaCompare(e.delta, delta);
      })[0];
      if (sourceEvent && sourceEvent.should) {
        if (event.cancelable) {
          event.preventDefault();
        }
        return;
      }
      if (!sourceEvent) {
        var shardNodes = (lastProps.current.shards || []).map(extractRef).filter(Boolean).filter(function(node) {
          return node.contains(event.target);
        });
        var shouldStop = shardNodes.length > 0 ? shouldCancelEvent(event, shardNodes[0]) : !lastProps.current.noIsolation;
        if (shouldStop) {
          if (event.cancelable) {
            event.preventDefault();
          }
        }
      }
    }, []);
    var shouldCancel = React27.useCallback(function(name, delta, target, should) {
      var event = { name, delta, target, should, shadowParent: getOutermostShadowParent(target) };
      shouldPreventQueue.current.push(event);
      setTimeout(function() {
        shouldPreventQueue.current = shouldPreventQueue.current.filter(function(e) {
          return e !== event;
        });
      }, 1);
    }, []);
    var scrollTouchStart = React27.useCallback(function(event) {
      touchStartRef.current = getTouchXY(event);
      activeAxis.current = void 0;
    }, []);
    var scrollWheel = React27.useCallback(function(event) {
      shouldCancel(event.type, getDeltaXY(event), event.target, shouldCancelEvent(event, props.lockRef.current));
    }, []);
    var scrollTouchMove = React27.useCallback(function(event) {
      shouldCancel(event.type, getTouchXY(event), event.target, shouldCancelEvent(event, props.lockRef.current));
    }, []);
    React27.useEffect(function() {
      lockStack.push(Style2);
      props.setCallbacks({
        onScrollCapture: scrollWheel,
        onWheelCapture: scrollWheel,
        onTouchMoveCapture: scrollTouchMove
      });
      document.addEventListener("wheel", shouldPrevent, nonPassive);
      document.addEventListener("touchmove", shouldPrevent, nonPassive);
      document.addEventListener("touchstart", scrollTouchStart, nonPassive);
      return function() {
        lockStack = lockStack.filter(function(inst) {
          return inst !== Style2;
        });
        document.removeEventListener("wheel", shouldPrevent, nonPassive);
        document.removeEventListener("touchmove", shouldPrevent, nonPassive);
        document.removeEventListener("touchstart", scrollTouchStart, nonPassive);
      };
    }, []);
    var removeScrollBar = props.removeScrollBar, inert = props.inert;
    return React27.createElement(
      React27.Fragment,
      null,
      inert ? React27.createElement(Style2, { styles: generateStyle(id) }) : null,
      removeScrollBar ? React27.createElement(RemoveScrollBar, { noRelative: props.noRelative, gapMode: props.gapMode }) : null
    );
  }
  function getOutermostShadowParent(node) {
    var shadowParent = null;
    while (node !== null) {
      if (node instanceof ShadowRoot) {
        shadowParent = node.host;
        node = node.host;
      }
      node = node.parentNode;
    }
    return shadowParent;
  }

  // node_modules/react-remove-scroll/dist/es2015/sidecar.js
  var sidecar_default = exportSidecar(effectCar, RemoveScrollSideCar);

  // node_modules/react-remove-scroll/dist/es2015/Combination.js
  var ReactRemoveScroll = React28.forwardRef(function(props, ref) {
    return React28.createElement(RemoveScroll, __assign({}, props, { ref, sideCar: sidecar_default }));
  });
  ReactRemoveScroll.classNames = RemoveScroll.classNames;
  var Combination_default = ReactRemoveScroll;

  // node_modules/aria-hidden/dist/es2015/index.js
  var getDefaultParent = function(originalTarget) {
    if (typeof document === "undefined") {
      return null;
    }
    var sampleTarget = Array.isArray(originalTarget) ? originalTarget[0] : originalTarget;
    return sampleTarget.ownerDocument.body;
  };
  var counterMap = /* @__PURE__ */ new WeakMap();
  var uncontrolledNodes = /* @__PURE__ */ new WeakMap();
  var markerMap = {};
  var lockCount = 0;
  var unwrapHost = function(node) {
    return node && (node.host || unwrapHost(node.parentNode));
  };
  var correctTargets = function(parent, targets) {
    return targets.map(function(target) {
      if (parent.contains(target)) {
        return target;
      }
      var correctedTarget = unwrapHost(target);
      if (correctedTarget && parent.contains(correctedTarget)) {
        return correctedTarget;
      }
      console.error("aria-hidden", target, "in not contained inside", parent, ". Doing nothing");
      return null;
    }).filter(function(x) {
      return Boolean(x);
    });
  };
  var applyAttributeToOthers = function(originalTarget, parentNode, markerName, controlAttribute) {
    var targets = correctTargets(parentNode, Array.isArray(originalTarget) ? originalTarget : [originalTarget]);
    if (!markerMap[markerName]) {
      markerMap[markerName] = /* @__PURE__ */ new WeakMap();
    }
    var markerCounter = markerMap[markerName];
    var hiddenNodes = [];
    var elementsToKeep = /* @__PURE__ */ new Set();
    var elementsToStop = new Set(targets);
    var keep = function(el) {
      if (!el || elementsToKeep.has(el)) {
        return;
      }
      elementsToKeep.add(el);
      keep(el.parentNode);
    };
    targets.forEach(keep);
    var deep = function(parent) {
      if (!parent || elementsToStop.has(parent)) {
        return;
      }
      Array.prototype.forEach.call(parent.children, function(node) {
        if (elementsToKeep.has(node)) {
          deep(node);
        } else {
          try {
            var attr = node.getAttribute(controlAttribute);
            var alreadyHidden = attr !== null && attr !== "false";
            var counterValue = (counterMap.get(node) || 0) + 1;
            var markerValue = (markerCounter.get(node) || 0) + 1;
            counterMap.set(node, counterValue);
            markerCounter.set(node, markerValue);
            hiddenNodes.push(node);
            if (counterValue === 1 && alreadyHidden) {
              uncontrolledNodes.set(node, true);
            }
            if (markerValue === 1) {
              node.setAttribute(markerName, "true");
            }
            if (!alreadyHidden) {
              node.setAttribute(controlAttribute, "true");
            }
          } catch (e) {
            console.error("aria-hidden: cannot operate on ", node, e);
          }
        }
      });
    };
    deep(parentNode);
    elementsToKeep.clear();
    lockCount++;
    return function() {
      hiddenNodes.forEach(function(node) {
        var counterValue = counterMap.get(node) - 1;
        var markerValue = markerCounter.get(node) - 1;
        counterMap.set(node, counterValue);
        markerCounter.set(node, markerValue);
        if (!counterValue) {
          if (!uncontrolledNodes.has(node)) {
            node.removeAttribute(controlAttribute);
          }
          uncontrolledNodes.delete(node);
        }
        if (!markerValue) {
          node.removeAttribute(markerName);
        }
      });
      lockCount--;
      if (!lockCount) {
        counterMap = /* @__PURE__ */ new WeakMap();
        counterMap = /* @__PURE__ */ new WeakMap();
        uncontrolledNodes = /* @__PURE__ */ new WeakMap();
        markerMap = {};
      }
    };
  };
  var hideOthers = function(originalTarget, parentNode, markerName) {
    if (markerName === void 0) {
      markerName = "data-aria-hidden";
    }
    var targets = Array.from(Array.isArray(originalTarget) ? originalTarget : [originalTarget]);
    var activeParentNode = parentNode || getDefaultParent(originalTarget);
    if (!activeParentNode) {
      return function() {
        return null;
      };
    }
    targets.push.apply(targets, Array.from(activeParentNode.querySelectorAll("[aria-live], script")));
    return applyAttributeToOthers(targets, activeParentNode, markerName, "aria-hidden");
  };

  // node_modules/@radix-ui/react-dialog/node_modules/@radix-ui/react-slot/dist/index.mjs
  var React29 = __toESM(require_react(), 1);
  var import_jsx_runtime12 = __toESM(require_jsx_runtime(), 1);
  // @__NO_SIDE_EFFECTS__
  function createSlot3(ownerName) {
    const SlotClone = /* @__PURE__ */ createSlotClone3(ownerName);
    const Slot22 = React29.forwardRef((props, forwardedRef) => {
      const { children, ...slotProps } = props;
      const childrenArray = React29.Children.toArray(children);
      const slottable = childrenArray.find(isSlottable3);
      if (slottable) {
        const newElement = slottable.props.children;
        const newChildren = childrenArray.map((child) => {
          if (child === slottable) {
            if (React29.Children.count(newElement) > 1) return React29.Children.only(null);
            return React29.isValidElement(newElement) ? newElement.props.children : null;
          } else {
            return child;
          }
        });
        return /* @__PURE__ */ (0, import_jsx_runtime12.jsx)(SlotClone, { ...slotProps, ref: forwardedRef, children: React29.isValidElement(newElement) ? React29.cloneElement(newElement, void 0, newChildren) : null });
      }
      return /* @__PURE__ */ (0, import_jsx_runtime12.jsx)(SlotClone, { ...slotProps, ref: forwardedRef, children });
    });
    Slot22.displayName = `${ownerName}.Slot`;
    return Slot22;
  }
  // @__NO_SIDE_EFFECTS__
  function createSlotClone3(ownerName) {
    const SlotClone = React29.forwardRef((props, forwardedRef) => {
      const { children, ...slotProps } = props;
      if (React29.isValidElement(children)) {
        const childrenRef = getElementRef4(children);
        const props2 = mergeProps3(slotProps, children.props);
        if (children.type !== React29.Fragment) {
          props2.ref = forwardedRef ? composeRefs(forwardedRef, childrenRef) : childrenRef;
        }
        return React29.cloneElement(children, props2);
      }
      return React29.Children.count(children) > 1 ? React29.Children.only(null) : null;
    });
    SlotClone.displayName = `${ownerName}.SlotClone`;
    return SlotClone;
  }
  var SLOTTABLE_IDENTIFIER3 = Symbol("radix.slottable");
  function isSlottable3(child) {
    return React29.isValidElement(child) && typeof child.type === "function" && "__radixId" in child.type && child.type.__radixId === SLOTTABLE_IDENTIFIER3;
  }
  function mergeProps3(slotProps, childProps) {
    const overrideProps = { ...childProps };
    for (const propName in childProps) {
      const slotPropValue = slotProps[propName];
      const childPropValue = childProps[propName];
      const isHandler = /^on[A-Z]/.test(propName);
      if (isHandler) {
        if (slotPropValue && childPropValue) {
          overrideProps[propName] = (...args) => {
            const result = childPropValue(...args);
            slotPropValue(...args);
            return result;
          };
        } else if (slotPropValue) {
          overrideProps[propName] = slotPropValue;
        }
      } else if (propName === "style") {
        overrideProps[propName] = { ...slotPropValue, ...childPropValue };
      } else if (propName === "className") {
        overrideProps[propName] = [slotPropValue, childPropValue].filter(Boolean).join(" ");
      }
    }
    return { ...slotProps, ...overrideProps };
  }
  function getElementRef4(element) {
    let getter = Object.getOwnPropertyDescriptor(element.props, "ref")?.get;
    let mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.ref;
    }
    getter = Object.getOwnPropertyDescriptor(element, "ref")?.get;
    mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.props.ref;
    }
    return element.props.ref || element.ref;
  }

  // node_modules/@radix-ui/react-dialog/dist/index.mjs
  var import_jsx_runtime13 = __toESM(require_jsx_runtime(), 1);
  var DIALOG_NAME = "Dialog";
  var [createDialogContext, createDialogScope] = createContextScope(DIALOG_NAME);
  var [DialogProvider, useDialogContext] = createDialogContext(DIALOG_NAME);
  var Dialog = (props) => {
    const {
      __scopeDialog,
      children,
      open: openProp,
      defaultOpen,
      onOpenChange,
      modal = true
    } = props;
    const triggerRef = React30.useRef(null);
    const contentRef = React30.useRef(null);
    const [open, setOpen] = useControllableState({
      prop: openProp,
      defaultProp: defaultOpen ?? false,
      onChange: onOpenChange,
      caller: DIALOG_NAME
    });
    return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(
      DialogProvider,
      {
        scope: __scopeDialog,
        triggerRef,
        contentRef,
        contentId: useId(),
        titleId: useId(),
        descriptionId: useId(),
        open,
        onOpenChange: setOpen,
        onOpenToggle: React30.useCallback(() => setOpen((prevOpen) => !prevOpen), [setOpen]),
        modal,
        children
      }
    );
  };
  Dialog.displayName = DIALOG_NAME;
  var TRIGGER_NAME = "DialogTrigger";
  var DialogTrigger = React30.forwardRef(
    (props, forwardedRef) => {
      const { __scopeDialog, ...triggerProps } = props;
      const context = useDialogContext(TRIGGER_NAME, __scopeDialog);
      const composedTriggerRef = useComposedRefs(forwardedRef, context.triggerRef);
      return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(
        Primitive.button,
        {
          type: "button",
          "aria-haspopup": "dialog",
          "aria-expanded": context.open,
          "aria-controls": context.contentId,
          "data-state": getState(context.open),
          ...triggerProps,
          ref: composedTriggerRef,
          onClick: composeEventHandlers(props.onClick, context.onOpenToggle)
        }
      );
    }
  );
  DialogTrigger.displayName = TRIGGER_NAME;
  var PORTAL_NAME2 = "DialogPortal";
  var [PortalProvider, usePortalContext] = createDialogContext(PORTAL_NAME2, {
    forceMount: void 0
  });
  var DialogPortal = (props) => {
    const { __scopeDialog, forceMount, children, container } = props;
    const context = useDialogContext(PORTAL_NAME2, __scopeDialog);
    return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(PortalProvider, { scope: __scopeDialog, forceMount, children: React30.Children.map(children, (child) => /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(Presence, { present: forceMount || context.open, children: /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(Portal, { asChild: true, container, children: child }) })) });
  };
  DialogPortal.displayName = PORTAL_NAME2;
  var OVERLAY_NAME = "DialogOverlay";
  var DialogOverlay = React30.forwardRef(
    (props, forwardedRef) => {
      const portalContext = usePortalContext(OVERLAY_NAME, props.__scopeDialog);
      const { forceMount = portalContext.forceMount, ...overlayProps } = props;
      const context = useDialogContext(OVERLAY_NAME, props.__scopeDialog);
      return context.modal ? /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(Presence, { present: forceMount || context.open, children: /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(DialogOverlayImpl, { ...overlayProps, ref: forwardedRef }) }) : null;
    }
  );
  DialogOverlay.displayName = OVERLAY_NAME;
  var Slot2 = createSlot3("DialogOverlay.RemoveScroll");
  var DialogOverlayImpl = React30.forwardRef(
    (props, forwardedRef) => {
      const { __scopeDialog, ...overlayProps } = props;
      const context = useDialogContext(OVERLAY_NAME, __scopeDialog);
      return (
        // Make sure `Content` is scrollable even when it doesn't live inside `RemoveScroll`
        // ie. when `Overlay` and `Content` are siblings
        /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(Combination_default, { as: Slot2, allowPinchZoom: true, shards: [context.contentRef], children: /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(
          Primitive.div,
          {
            "data-state": getState(context.open),
            ...overlayProps,
            ref: forwardedRef,
            style: { pointerEvents: "auto", ...overlayProps.style }
          }
        ) })
      );
    }
  );
  var CONTENT_NAME = "DialogContent";
  var DialogContent = React30.forwardRef(
    (props, forwardedRef) => {
      const portalContext = usePortalContext(CONTENT_NAME, props.__scopeDialog);
      const { forceMount = portalContext.forceMount, ...contentProps } = props;
      const context = useDialogContext(CONTENT_NAME, props.__scopeDialog);
      return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(Presence, { present: forceMount || context.open, children: context.modal ? /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(DialogContentModal, { ...contentProps, ref: forwardedRef }) : /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(DialogContentNonModal, { ...contentProps, ref: forwardedRef }) });
    }
  );
  DialogContent.displayName = CONTENT_NAME;
  var DialogContentModal = React30.forwardRef(
    (props, forwardedRef) => {
      const context = useDialogContext(CONTENT_NAME, props.__scopeDialog);
      const contentRef = React30.useRef(null);
      const composedRefs = useComposedRefs(forwardedRef, context.contentRef, contentRef);
      React30.useEffect(() => {
        const content = contentRef.current;
        if (content) return hideOthers(content);
      }, []);
      return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(
        DialogContentImpl,
        {
          ...props,
          ref: composedRefs,
          trapFocus: context.open,
          disableOutsidePointerEvents: true,
          onCloseAutoFocus: composeEventHandlers(props.onCloseAutoFocus, (event) => {
            event.preventDefault();
            context.triggerRef.current?.focus();
          }),
          onPointerDownOutside: composeEventHandlers(props.onPointerDownOutside, (event) => {
            const originalEvent = event.detail.originalEvent;
            const ctrlLeftClick = originalEvent.button === 0 && originalEvent.ctrlKey === true;
            const isRightClick = originalEvent.button === 2 || ctrlLeftClick;
            if (isRightClick) event.preventDefault();
          }),
          onFocusOutside: composeEventHandlers(
            props.onFocusOutside,
            (event) => event.preventDefault()
          )
        }
      );
    }
  );
  var DialogContentNonModal = React30.forwardRef(
    (props, forwardedRef) => {
      const context = useDialogContext(CONTENT_NAME, props.__scopeDialog);
      const hasInteractedOutsideRef = React30.useRef(false);
      const hasPointerDownOutsideRef = React30.useRef(false);
      return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(
        DialogContentImpl,
        {
          ...props,
          ref: forwardedRef,
          trapFocus: false,
          disableOutsidePointerEvents: false,
          onCloseAutoFocus: (event) => {
            props.onCloseAutoFocus?.(event);
            if (!event.defaultPrevented) {
              if (!hasInteractedOutsideRef.current) context.triggerRef.current?.focus();
              event.preventDefault();
            }
            hasInteractedOutsideRef.current = false;
            hasPointerDownOutsideRef.current = false;
          },
          onInteractOutside: (event) => {
            props.onInteractOutside?.(event);
            if (!event.defaultPrevented) {
              hasInteractedOutsideRef.current = true;
              if (event.detail.originalEvent.type === "pointerdown") {
                hasPointerDownOutsideRef.current = true;
              }
            }
            const target = event.target;
            const targetIsTrigger = context.triggerRef.current?.contains(target);
            if (targetIsTrigger) event.preventDefault();
            if (event.detail.originalEvent.type === "focusin" && hasPointerDownOutsideRef.current) {
              event.preventDefault();
            }
          }
        }
      );
    }
  );
  var DialogContentImpl = React30.forwardRef(
    (props, forwardedRef) => {
      const { __scopeDialog, trapFocus, onOpenAutoFocus, onCloseAutoFocus, ...contentProps } = props;
      const context = useDialogContext(CONTENT_NAME, __scopeDialog);
      const contentRef = React30.useRef(null);
      const composedRefs = useComposedRefs(forwardedRef, contentRef);
      useFocusGuards();
      return /* @__PURE__ */ (0, import_jsx_runtime13.jsxs)(import_jsx_runtime13.Fragment, { children: [
        /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(
          FocusScope,
          {
            asChild: true,
            loop: true,
            trapped: trapFocus,
            onMountAutoFocus: onOpenAutoFocus,
            onUnmountAutoFocus: onCloseAutoFocus,
            children: /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(
              DismissableLayer,
              {
                role: "dialog",
                id: context.contentId,
                "aria-describedby": context.descriptionId,
                "aria-labelledby": context.titleId,
                "data-state": getState(context.open),
                ...contentProps,
                ref: composedRefs,
                onDismiss: () => context.onOpenChange(false)
              }
            )
          }
        ),
        /* @__PURE__ */ (0, import_jsx_runtime13.jsxs)(import_jsx_runtime13.Fragment, { children: [
          /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(TitleWarning, { titleId: context.titleId }),
          /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(DescriptionWarning, { contentRef, descriptionId: context.descriptionId })
        ] })
      ] });
    }
  );
  var TITLE_NAME = "DialogTitle";
  var DialogTitle = React30.forwardRef(
    (props, forwardedRef) => {
      const { __scopeDialog, ...titleProps } = props;
      const context = useDialogContext(TITLE_NAME, __scopeDialog);
      return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(Primitive.h2, { id: context.titleId, ...titleProps, ref: forwardedRef });
    }
  );
  DialogTitle.displayName = TITLE_NAME;
  var DESCRIPTION_NAME = "DialogDescription";
  var DialogDescription = React30.forwardRef(
    (props, forwardedRef) => {
      const { __scopeDialog, ...descriptionProps } = props;
      const context = useDialogContext(DESCRIPTION_NAME, __scopeDialog);
      return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(Primitive.p, { id: context.descriptionId, ...descriptionProps, ref: forwardedRef });
    }
  );
  DialogDescription.displayName = DESCRIPTION_NAME;
  var CLOSE_NAME = "DialogClose";
  var DialogClose = React30.forwardRef(
    (props, forwardedRef) => {
      const { __scopeDialog, ...closeProps } = props;
      const context = useDialogContext(CLOSE_NAME, __scopeDialog);
      return /* @__PURE__ */ (0, import_jsx_runtime13.jsx)(
        Primitive.button,
        {
          type: "button",
          ...closeProps,
          ref: forwardedRef,
          onClick: composeEventHandlers(props.onClick, () => context.onOpenChange(false))
        }
      );
    }
  );
  DialogClose.displayName = CLOSE_NAME;
  function getState(open) {
    return open ? "open" : "closed";
  }
  var TITLE_WARNING_NAME = "DialogTitleWarning";
  var [WarningProvider, useWarningContext] = createContext2(TITLE_WARNING_NAME, {
    contentName: CONTENT_NAME,
    titleName: TITLE_NAME,
    docsSlug: "dialog"
  });
  var TitleWarning = ({ titleId }) => {
    const titleWarningContext = useWarningContext(TITLE_WARNING_NAME);
    const MESSAGE = `\`${titleWarningContext.contentName}\` requires a \`${titleWarningContext.titleName}\` for the component to be accessible for screen reader users.

If you want to hide the \`${titleWarningContext.titleName}\`, you can wrap it with our VisuallyHidden component.

For more information, see https://radix-ui.com/primitives/docs/components/${titleWarningContext.docsSlug}`;
    React30.useEffect(() => {
      if (titleId) {
        const hasTitle = document.getElementById(titleId);
        if (!hasTitle) console.error(MESSAGE);
      }
    }, [MESSAGE, titleId]);
    return null;
  };
  var DESCRIPTION_WARNING_NAME = "DialogDescriptionWarning";
  var DescriptionWarning = ({ contentRef, descriptionId }) => {
    const descriptionWarningContext = useWarningContext(DESCRIPTION_WARNING_NAME);
    const MESSAGE = `Warning: Missing \`Description\` or \`aria-describedby={undefined}\` for {${descriptionWarningContext.contentName}}.`;
    React30.useEffect(() => {
      const describedById = contentRef.current?.getAttribute("aria-describedby");
      if (descriptionId && describedById) {
        const hasDescription = document.getElementById(descriptionId);
        if (!hasDescription) console.warn(MESSAGE);
      }
    }, [MESSAGE, contentRef, descriptionId]);
    return null;
  };
  var Root = Dialog;
  var Portal2 = DialogPortal;
  var Overlay = DialogOverlay;
  var Content = DialogContent;
  var Title = DialogTitle;
  var Description = DialogDescription;

  // src/components/ui/dialog.jsx
  var import_jsx_runtime14 = __toESM(require_jsx_runtime(), 1);
  var Dialog2 = Root;
  var DialogPortal2 = Portal2;
  function DialogOverlay2({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime14.jsx)(Overlay, { className: cn("ui-dialog__overlay", className), ...props });
  }
  function DialogContent2({ className, children, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime14.jsxs)(DialogPortal2, { children: [
      /* @__PURE__ */ (0, import_jsx_runtime14.jsx)(DialogOverlay2, {}),
      /* @__PURE__ */ (0, import_jsx_runtime14.jsx)(Content, { className: cn("ui-dialog__content panel", className), ...props, children })
    ] });
  }
  function DialogHeader({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime14.jsx)("div", { className: cn("ui-dialog__header", className), ...props });
  }
  function DialogTitle2({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime14.jsx)(Title, { className: cn("ui-dialog__title", className), ...props });
  }
  function DialogDescription2({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime14.jsx)(Description, { className: cn("ui-dialog__description", className), ...props });
  }

  // node_modules/@radix-ui/react-accordion/dist/index.mjs
  var import_react8 = __toESM(require_react(), 1);

  // node_modules/@radix-ui/react-collection/dist/index.mjs
  var import_react6 = __toESM(require_react(), 1);

  // node_modules/@radix-ui/react-collection/node_modules/@radix-ui/react-slot/dist/index.mjs
  var React31 = __toESM(require_react(), 1);
  var import_jsx_runtime15 = __toESM(require_jsx_runtime(), 1);
  // @__NO_SIDE_EFFECTS__
  function createSlot4(ownerName) {
    const SlotClone = /* @__PURE__ */ createSlotClone4(ownerName);
    const Slot22 = React31.forwardRef((props, forwardedRef) => {
      const { children, ...slotProps } = props;
      const childrenArray = React31.Children.toArray(children);
      const slottable = childrenArray.find(isSlottable4);
      if (slottable) {
        const newElement = slottable.props.children;
        const newChildren = childrenArray.map((child) => {
          if (child === slottable) {
            if (React31.Children.count(newElement) > 1) return React31.Children.only(null);
            return React31.isValidElement(newElement) ? newElement.props.children : null;
          } else {
            return child;
          }
        });
        return /* @__PURE__ */ (0, import_jsx_runtime15.jsx)(SlotClone, { ...slotProps, ref: forwardedRef, children: React31.isValidElement(newElement) ? React31.cloneElement(newElement, void 0, newChildren) : null });
      }
      return /* @__PURE__ */ (0, import_jsx_runtime15.jsx)(SlotClone, { ...slotProps, ref: forwardedRef, children });
    });
    Slot22.displayName = `${ownerName}.Slot`;
    return Slot22;
  }
  // @__NO_SIDE_EFFECTS__
  function createSlotClone4(ownerName) {
    const SlotClone = React31.forwardRef((props, forwardedRef) => {
      const { children, ...slotProps } = props;
      if (React31.isValidElement(children)) {
        const childrenRef = getElementRef5(children);
        const props2 = mergeProps4(slotProps, children.props);
        if (children.type !== React31.Fragment) {
          props2.ref = forwardedRef ? composeRefs(forwardedRef, childrenRef) : childrenRef;
        }
        return React31.cloneElement(children, props2);
      }
      return React31.Children.count(children) > 1 ? React31.Children.only(null) : null;
    });
    SlotClone.displayName = `${ownerName}.SlotClone`;
    return SlotClone;
  }
  var SLOTTABLE_IDENTIFIER4 = Symbol("radix.slottable");
  function isSlottable4(child) {
    return React31.isValidElement(child) && typeof child.type === "function" && "__radixId" in child.type && child.type.__radixId === SLOTTABLE_IDENTIFIER4;
  }
  function mergeProps4(slotProps, childProps) {
    const overrideProps = { ...childProps };
    for (const propName in childProps) {
      const slotPropValue = slotProps[propName];
      const childPropValue = childProps[propName];
      const isHandler = /^on[A-Z]/.test(propName);
      if (isHandler) {
        if (slotPropValue && childPropValue) {
          overrideProps[propName] = (...args) => {
            const result = childPropValue(...args);
            slotPropValue(...args);
            return result;
          };
        } else if (slotPropValue) {
          overrideProps[propName] = slotPropValue;
        }
      } else if (propName === "style") {
        overrideProps[propName] = { ...slotPropValue, ...childPropValue };
      } else if (propName === "className") {
        overrideProps[propName] = [slotPropValue, childPropValue].filter(Boolean).join(" ");
      }
    }
    return { ...slotProps, ...overrideProps };
  }
  function getElementRef5(element) {
    let getter = Object.getOwnPropertyDescriptor(element.props, "ref")?.get;
    let mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.ref;
    }
    getter = Object.getOwnPropertyDescriptor(element, "ref")?.get;
    mayWarn = getter && "isReactWarning" in getter && getter.isReactWarning;
    if (mayWarn) {
      return element.props.ref;
    }
    return element.props.ref || element.ref;
  }

  // node_modules/@radix-ui/react-collection/dist/index.mjs
  var import_jsx_runtime16 = __toESM(require_jsx_runtime(), 1);
  var import_react7 = __toESM(require_react(), 1);
  var import_jsx_runtime17 = __toESM(require_jsx_runtime(), 1);
  function createCollection(name) {
    const PROVIDER_NAME = name + "CollectionProvider";
    const [createCollectionContext, createCollectionScope3] = createContextScope(PROVIDER_NAME);
    const [CollectionProviderImpl, useCollectionContext] = createCollectionContext(
      PROVIDER_NAME,
      { collectionRef: { current: null }, itemMap: /* @__PURE__ */ new Map() }
    );
    const CollectionProvider = (props) => {
      const { scope, children } = props;
      const ref = import_react6.default.useRef(null);
      const itemMap = import_react6.default.useRef(/* @__PURE__ */ new Map()).current;
      return /* @__PURE__ */ (0, import_jsx_runtime16.jsx)(CollectionProviderImpl, { scope, itemMap, collectionRef: ref, children });
    };
    CollectionProvider.displayName = PROVIDER_NAME;
    const COLLECTION_SLOT_NAME = name + "CollectionSlot";
    const CollectionSlotImpl = createSlot4(COLLECTION_SLOT_NAME);
    const CollectionSlot = import_react6.default.forwardRef(
      (props, forwardedRef) => {
        const { scope, children } = props;
        const context = useCollectionContext(COLLECTION_SLOT_NAME, scope);
        const composedRefs = useComposedRefs(forwardedRef, context.collectionRef);
        return /* @__PURE__ */ (0, import_jsx_runtime16.jsx)(CollectionSlotImpl, { ref: composedRefs, children });
      }
    );
    CollectionSlot.displayName = COLLECTION_SLOT_NAME;
    const ITEM_SLOT_NAME = name + "CollectionItemSlot";
    const ITEM_DATA_ATTR = "data-radix-collection-item";
    const CollectionItemSlotImpl = createSlot4(ITEM_SLOT_NAME);
    const CollectionItemSlot = import_react6.default.forwardRef(
      (props, forwardedRef) => {
        const { scope, children, ...itemData } = props;
        const ref = import_react6.default.useRef(null);
        const composedRefs = useComposedRefs(forwardedRef, ref);
        const context = useCollectionContext(ITEM_SLOT_NAME, scope);
        import_react6.default.useEffect(() => {
          context.itemMap.set(ref, { ref, ...itemData });
          return () => void context.itemMap.delete(ref);
        });
        return /* @__PURE__ */ (0, import_jsx_runtime16.jsx)(CollectionItemSlotImpl, { ...{ [ITEM_DATA_ATTR]: "" }, ref: composedRefs, children });
      }
    );
    CollectionItemSlot.displayName = ITEM_SLOT_NAME;
    function useCollection3(scope) {
      const context = useCollectionContext(name + "CollectionConsumer", scope);
      const getItems = import_react6.default.useCallback(() => {
        const collectionNode = context.collectionRef.current;
        if (!collectionNode) return [];
        const orderedNodes = Array.from(collectionNode.querySelectorAll(`[${ITEM_DATA_ATTR}]`));
        const items = Array.from(context.itemMap.values());
        const orderedItems = items.sort(
          (a, b) => orderedNodes.indexOf(a.ref.current) - orderedNodes.indexOf(b.ref.current)
        );
        return orderedItems;
      }, [context.collectionRef, context.itemMap]);
      return getItems;
    }
    return [
      { Provider: CollectionProvider, Slot: CollectionSlot, ItemSlot: CollectionItemSlot },
      useCollection3,
      createCollectionScope3
    ];
  }

  // node_modules/@radix-ui/react-collapsible/dist/index.mjs
  var React33 = __toESM(require_react(), 1);
  var import_jsx_runtime18 = __toESM(require_jsx_runtime(), 1);
  var COLLAPSIBLE_NAME = "Collapsible";
  var [createCollapsibleContext, createCollapsibleScope] = createContextScope(COLLAPSIBLE_NAME);
  var [CollapsibleProvider, useCollapsibleContext] = createCollapsibleContext(COLLAPSIBLE_NAME);
  var Collapsible = React33.forwardRef(
    (props, forwardedRef) => {
      const {
        __scopeCollapsible,
        open: openProp,
        defaultOpen,
        disabled,
        onOpenChange,
        ...collapsibleProps
      } = props;
      const [open, setOpen] = useControllableState({
        prop: openProp,
        defaultProp: defaultOpen ?? false,
        onChange: onOpenChange,
        caller: COLLAPSIBLE_NAME
      });
      return /* @__PURE__ */ (0, import_jsx_runtime18.jsx)(
        CollapsibleProvider,
        {
          scope: __scopeCollapsible,
          disabled,
          contentId: useId(),
          open,
          onOpenToggle: React33.useCallback(() => setOpen((prevOpen) => !prevOpen), [setOpen]),
          children: /* @__PURE__ */ (0, import_jsx_runtime18.jsx)(
            Primitive.div,
            {
              "data-state": getState2(open),
              "data-disabled": disabled ? "" : void 0,
              ...collapsibleProps,
              ref: forwardedRef
            }
          )
        }
      );
    }
  );
  Collapsible.displayName = COLLAPSIBLE_NAME;
  var TRIGGER_NAME2 = "CollapsibleTrigger";
  var CollapsibleTrigger = React33.forwardRef(
    (props, forwardedRef) => {
      const { __scopeCollapsible, ...triggerProps } = props;
      const context = useCollapsibleContext(TRIGGER_NAME2, __scopeCollapsible);
      return /* @__PURE__ */ (0, import_jsx_runtime18.jsx)(
        Primitive.button,
        {
          type: "button",
          "aria-controls": context.contentId,
          "aria-expanded": context.open || false,
          "data-state": getState2(context.open),
          "data-disabled": context.disabled ? "" : void 0,
          disabled: context.disabled,
          ...triggerProps,
          ref: forwardedRef,
          onClick: composeEventHandlers(props.onClick, context.onOpenToggle)
        }
      );
    }
  );
  CollapsibleTrigger.displayName = TRIGGER_NAME2;
  var CONTENT_NAME2 = "CollapsibleContent";
  var CollapsibleContent = React33.forwardRef(
    (props, forwardedRef) => {
      const { forceMount, ...contentProps } = props;
      const context = useCollapsibleContext(CONTENT_NAME2, props.__scopeCollapsible);
      return /* @__PURE__ */ (0, import_jsx_runtime18.jsx)(Presence, { present: forceMount || context.open, children: ({ present }) => /* @__PURE__ */ (0, import_jsx_runtime18.jsx)(CollapsibleContentImpl, { ...contentProps, ref: forwardedRef, present }) });
    }
  );
  CollapsibleContent.displayName = CONTENT_NAME2;
  var CollapsibleContentImpl = React33.forwardRef((props, forwardedRef) => {
    const { __scopeCollapsible, present, children, ...contentProps } = props;
    const context = useCollapsibleContext(CONTENT_NAME2, __scopeCollapsible);
    const [isPresent, setIsPresent] = React33.useState(present);
    const ref = React33.useRef(null);
    const composedRefs = useComposedRefs(forwardedRef, ref);
    const heightRef = React33.useRef(0);
    const height = heightRef.current;
    const widthRef = React33.useRef(0);
    const width = widthRef.current;
    const isOpen = context.open || isPresent;
    const isMountAnimationPreventedRef = React33.useRef(isOpen);
    const originalStylesRef = React33.useRef(void 0);
    React33.useEffect(() => {
      const rAF = requestAnimationFrame(() => isMountAnimationPreventedRef.current = false);
      return () => cancelAnimationFrame(rAF);
    }, []);
    useLayoutEffect2(() => {
      const node = ref.current;
      if (node) {
        originalStylesRef.current = originalStylesRef.current || {
          transitionDuration: node.style.transitionDuration,
          animationName: node.style.animationName
        };
        node.style.transitionDuration = "0s";
        node.style.animationName = "none";
        const rect = node.getBoundingClientRect();
        heightRef.current = rect.height;
        widthRef.current = rect.width;
        if (!isMountAnimationPreventedRef.current) {
          node.style.transitionDuration = originalStylesRef.current.transitionDuration;
          node.style.animationName = originalStylesRef.current.animationName;
        }
        setIsPresent(present);
      }
    }, [context.open, present]);
    return /* @__PURE__ */ (0, import_jsx_runtime18.jsx)(
      Primitive.div,
      {
        "data-state": getState2(context.open),
        "data-disabled": context.disabled ? "" : void 0,
        id: context.contentId,
        hidden: !isOpen,
        ...contentProps,
        ref: composedRefs,
        style: {
          [`--radix-collapsible-content-height`]: height ? `${height}px` : void 0,
          [`--radix-collapsible-content-width`]: width ? `${width}px` : void 0,
          ...props.style
        },
        children: isOpen && children
      }
    );
  });
  function getState2(open) {
    return open ? "open" : "closed";
  }
  var Root2 = Collapsible;
  var Trigger2 = CollapsibleTrigger;
  var Content2 = CollapsibleContent;

  // node_modules/@radix-ui/react-direction/dist/index.mjs
  var React34 = __toESM(require_react(), 1);
  var import_jsx_runtime19 = __toESM(require_jsx_runtime(), 1);
  var DirectionContext = React34.createContext(void 0);
  function useDirection(localDir) {
    const globalDir = React34.useContext(DirectionContext);
    return localDir || globalDir || "ltr";
  }

  // node_modules/@radix-ui/react-accordion/dist/index.mjs
  var import_jsx_runtime20 = __toESM(require_jsx_runtime(), 1);
  var ACCORDION_NAME = "Accordion";
  var ACCORDION_KEYS = ["Home", "End", "ArrowDown", "ArrowUp", "ArrowLeft", "ArrowRight"];
  var [Collection, useCollection, createCollectionScope] = createCollection(ACCORDION_NAME);
  var [createAccordionContext, createAccordionScope] = createContextScope(ACCORDION_NAME, [
    createCollectionScope,
    createCollapsibleScope
  ]);
  var useCollapsibleScope = createCollapsibleScope();
  var Accordion = import_react8.default.forwardRef(
    (props, forwardedRef) => {
      const { type, ...accordionProps } = props;
      const singleProps = accordionProps;
      const multipleProps = accordionProps;
      return /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(Collection.Provider, { scope: props.__scopeAccordion, children: type === "multiple" ? /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(AccordionImplMultiple, { ...multipleProps, ref: forwardedRef }) : /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(AccordionImplSingle, { ...singleProps, ref: forwardedRef }) });
    }
  );
  Accordion.displayName = ACCORDION_NAME;
  var [AccordionValueProvider, useAccordionValueContext] = createAccordionContext(ACCORDION_NAME);
  var [AccordionCollapsibleProvider, useAccordionCollapsibleContext] = createAccordionContext(
    ACCORDION_NAME,
    { collapsible: false }
  );
  var AccordionImplSingle = import_react8.default.forwardRef(
    (props, forwardedRef) => {
      const {
        value: valueProp,
        defaultValue,
        onValueChange = () => {
        },
        collapsible = false,
        ...accordionSingleProps
      } = props;
      const [value, setValue] = useControllableState({
        prop: valueProp,
        defaultProp: defaultValue ?? "",
        onChange: onValueChange,
        caller: ACCORDION_NAME
      });
      return /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
        AccordionValueProvider,
        {
          scope: props.__scopeAccordion,
          value: import_react8.default.useMemo(() => value ? [value] : [], [value]),
          onItemOpen: setValue,
          onItemClose: import_react8.default.useCallback(() => collapsible && setValue(""), [collapsible, setValue]),
          children: /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(AccordionCollapsibleProvider, { scope: props.__scopeAccordion, collapsible, children: /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(AccordionImpl, { ...accordionSingleProps, ref: forwardedRef }) })
        }
      );
    }
  );
  var AccordionImplMultiple = import_react8.default.forwardRef((props, forwardedRef) => {
    const {
      value: valueProp,
      defaultValue,
      onValueChange = () => {
      },
      ...accordionMultipleProps
    } = props;
    const [value, setValue] = useControllableState({
      prop: valueProp,
      defaultProp: defaultValue ?? [],
      onChange: onValueChange,
      caller: ACCORDION_NAME
    });
    const handleItemOpen = import_react8.default.useCallback(
      (itemValue) => setValue((prevValue = []) => [...prevValue, itemValue]),
      [setValue]
    );
    const handleItemClose = import_react8.default.useCallback(
      (itemValue) => setValue((prevValue = []) => prevValue.filter((value2) => value2 !== itemValue)),
      [setValue]
    );
    return /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
      AccordionValueProvider,
      {
        scope: props.__scopeAccordion,
        value,
        onItemOpen: handleItemOpen,
        onItemClose: handleItemClose,
        children: /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(AccordionCollapsibleProvider, { scope: props.__scopeAccordion, collapsible: true, children: /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(AccordionImpl, { ...accordionMultipleProps, ref: forwardedRef }) })
      }
    );
  });
  var [AccordionImplProvider, useAccordionContext] = createAccordionContext(ACCORDION_NAME);
  var AccordionImpl = import_react8.default.forwardRef(
    (props, forwardedRef) => {
      const { __scopeAccordion, disabled, dir, orientation = "vertical", ...accordionProps } = props;
      const accordionRef = import_react8.default.useRef(null);
      const composedRefs = useComposedRefs(accordionRef, forwardedRef);
      const getItems = useCollection(__scopeAccordion);
      const direction = useDirection(dir);
      const isDirectionLTR = direction === "ltr";
      const handleKeyDown = composeEventHandlers(props.onKeyDown, (event) => {
        if (!ACCORDION_KEYS.includes(event.key)) return;
        const target = event.target;
        const triggerCollection = getItems().filter((item) => !item.ref.current?.disabled);
        const triggerIndex = triggerCollection.findIndex((item) => item.ref.current === target);
        const triggerCount = triggerCollection.length;
        if (triggerIndex === -1) return;
        event.preventDefault();
        let nextIndex = triggerIndex;
        const homeIndex = 0;
        const endIndex = triggerCount - 1;
        const moveNext = () => {
          nextIndex = triggerIndex + 1;
          if (nextIndex > endIndex) {
            nextIndex = homeIndex;
          }
        };
        const movePrev = () => {
          nextIndex = triggerIndex - 1;
          if (nextIndex < homeIndex) {
            nextIndex = endIndex;
          }
        };
        switch (event.key) {
          case "Home":
            nextIndex = homeIndex;
            break;
          case "End":
            nextIndex = endIndex;
            break;
          case "ArrowRight":
            if (orientation === "horizontal") {
              if (isDirectionLTR) {
                moveNext();
              } else {
                movePrev();
              }
            }
            break;
          case "ArrowDown":
            if (orientation === "vertical") {
              moveNext();
            }
            break;
          case "ArrowLeft":
            if (orientation === "horizontal") {
              if (isDirectionLTR) {
                movePrev();
              } else {
                moveNext();
              }
            }
            break;
          case "ArrowUp":
            if (orientation === "vertical") {
              movePrev();
            }
            break;
        }
        const clampedIndex = nextIndex % triggerCount;
        triggerCollection[clampedIndex].ref.current?.focus();
      });
      return /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
        AccordionImplProvider,
        {
          scope: __scopeAccordion,
          disabled,
          direction: dir,
          orientation,
          children: /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(Collection.Slot, { scope: __scopeAccordion, children: /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
            Primitive.div,
            {
              ...accordionProps,
              "data-orientation": orientation,
              ref: composedRefs,
              onKeyDown: disabled ? void 0 : handleKeyDown
            }
          ) })
        }
      );
    }
  );
  var ITEM_NAME = "AccordionItem";
  var [AccordionItemProvider, useAccordionItemContext] = createAccordionContext(ITEM_NAME);
  var AccordionItem = import_react8.default.forwardRef(
    (props, forwardedRef) => {
      const { __scopeAccordion, value, ...accordionItemProps } = props;
      const accordionContext = useAccordionContext(ITEM_NAME, __scopeAccordion);
      const valueContext = useAccordionValueContext(ITEM_NAME, __scopeAccordion);
      const collapsibleScope = useCollapsibleScope(__scopeAccordion);
      const triggerId = useId();
      const open = value && valueContext.value.includes(value) || false;
      const disabled = accordionContext.disabled || props.disabled;
      return /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
        AccordionItemProvider,
        {
          scope: __scopeAccordion,
          open,
          disabled,
          triggerId,
          children: /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
            Root2,
            {
              "data-orientation": accordionContext.orientation,
              "data-state": getState3(open),
              ...collapsibleScope,
              ...accordionItemProps,
              ref: forwardedRef,
              disabled,
              open,
              onOpenChange: (open2) => {
                if (open2) {
                  valueContext.onItemOpen(value);
                } else {
                  valueContext.onItemClose(value);
                }
              }
            }
          )
        }
      );
    }
  );
  AccordionItem.displayName = ITEM_NAME;
  var HEADER_NAME = "AccordionHeader";
  var AccordionHeader = import_react8.default.forwardRef(
    (props, forwardedRef) => {
      const { __scopeAccordion, ...headerProps } = props;
      const accordionContext = useAccordionContext(ACCORDION_NAME, __scopeAccordion);
      const itemContext = useAccordionItemContext(HEADER_NAME, __scopeAccordion);
      return /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
        Primitive.h3,
        {
          "data-orientation": accordionContext.orientation,
          "data-state": getState3(itemContext.open),
          "data-disabled": itemContext.disabled ? "" : void 0,
          ...headerProps,
          ref: forwardedRef
        }
      );
    }
  );
  AccordionHeader.displayName = HEADER_NAME;
  var TRIGGER_NAME3 = "AccordionTrigger";
  var AccordionTrigger = import_react8.default.forwardRef(
    (props, forwardedRef) => {
      const { __scopeAccordion, ...triggerProps } = props;
      const accordionContext = useAccordionContext(ACCORDION_NAME, __scopeAccordion);
      const itemContext = useAccordionItemContext(TRIGGER_NAME3, __scopeAccordion);
      const collapsibleContext = useAccordionCollapsibleContext(TRIGGER_NAME3, __scopeAccordion);
      const collapsibleScope = useCollapsibleScope(__scopeAccordion);
      return /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(Collection.ItemSlot, { scope: __scopeAccordion, children: /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
        Trigger2,
        {
          "aria-disabled": itemContext.open && !collapsibleContext.collapsible || void 0,
          "data-orientation": accordionContext.orientation,
          id: itemContext.triggerId,
          ...collapsibleScope,
          ...triggerProps,
          ref: forwardedRef
        }
      ) });
    }
  );
  AccordionTrigger.displayName = TRIGGER_NAME3;
  var CONTENT_NAME3 = "AccordionContent";
  var AccordionContent = import_react8.default.forwardRef(
    (props, forwardedRef) => {
      const { __scopeAccordion, ...contentProps } = props;
      const accordionContext = useAccordionContext(ACCORDION_NAME, __scopeAccordion);
      const itemContext = useAccordionItemContext(CONTENT_NAME3, __scopeAccordion);
      const collapsibleScope = useCollapsibleScope(__scopeAccordion);
      return /* @__PURE__ */ (0, import_jsx_runtime20.jsx)(
        Content2,
        {
          role: "region",
          "aria-labelledby": itemContext.triggerId,
          "data-orientation": accordionContext.orientation,
          ...collapsibleScope,
          ...contentProps,
          ref: forwardedRef,
          style: {
            ["--radix-accordion-content-height"]: "var(--radix-collapsible-content-height)",
            ["--radix-accordion-content-width"]: "var(--radix-collapsible-content-width)",
            ...props.style
          }
        }
      );
    }
  );
  AccordionContent.displayName = CONTENT_NAME3;
  function getState3(open) {
    return open ? "open" : "closed";
  }
  var Root22 = Accordion;
  var Item = AccordionItem;
  var Header = AccordionHeader;
  var Trigger22 = AccordionTrigger;
  var Content22 = AccordionContent;

  // src/components/ui/accordion.jsx
  var import_jsx_runtime21 = __toESM(require_jsx_runtime(), 1);
  var Accordion2 = Root22;
  function AccordionItem2({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime21.jsx)(Item, { className: cn("ui-accordion__item", className), ...props });
  }
  function AccordionTrigger2({ className, children, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime21.jsx)(Header, { className: "ui-accordion__header", children: /* @__PURE__ */ (0, import_jsx_runtime21.jsx)(Trigger22, { className: cn("ui-accordion__trigger", className), ...props, children }) });
  }
  function AccordionContent2({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime21.jsx)(Content22, { className: cn("ui-accordion__content", className), ...props });
  }

  // node_modules/@radix-ui/react-scroll-area/dist/index.mjs
  var React211 = __toESM(require_react(), 1);

  // node_modules/@radix-ui/number/dist/index.mjs
  function clamp(value, [min, max]) {
    return Math.min(max, Math.max(min, value));
  }

  // node_modules/@radix-ui/react-scroll-area/dist/index.mjs
  var React36 = __toESM(require_react(), 1);
  var import_jsx_runtime22 = __toESM(require_jsx_runtime(), 1);
  function useStateMachine2(initialState, machine) {
    return React36.useReducer((state, event) => {
      const nextState = machine[state][event];
      return nextState ?? state;
    }, initialState);
  }
  var SCROLL_AREA_NAME = "ScrollArea";
  var [createScrollAreaContext, createScrollAreaScope] = createContextScope(SCROLL_AREA_NAME);
  var [ScrollAreaProvider, useScrollAreaContext] = createScrollAreaContext(SCROLL_AREA_NAME);
  var ScrollArea = React211.forwardRef(
    (props, forwardedRef) => {
      const {
        __scopeScrollArea,
        type = "hover",
        dir,
        scrollHideDelay = 600,
        ...scrollAreaProps
      } = props;
      const [scrollArea, setScrollArea] = React211.useState(null);
      const [viewport, setViewport] = React211.useState(null);
      const [content, setContent] = React211.useState(null);
      const [scrollbarX, setScrollbarX] = React211.useState(null);
      const [scrollbarY, setScrollbarY] = React211.useState(null);
      const [cornerWidth, setCornerWidth] = React211.useState(0);
      const [cornerHeight, setCornerHeight] = React211.useState(0);
      const [scrollbarXEnabled, setScrollbarXEnabled] = React211.useState(false);
      const [scrollbarYEnabled, setScrollbarYEnabled] = React211.useState(false);
      const composedRefs = useComposedRefs(forwardedRef, (node) => setScrollArea(node));
      const direction = useDirection(dir);
      return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
        ScrollAreaProvider,
        {
          scope: __scopeScrollArea,
          type,
          dir: direction,
          scrollHideDelay,
          scrollArea,
          viewport,
          onViewportChange: setViewport,
          content,
          onContentChange: setContent,
          scrollbarX,
          onScrollbarXChange: setScrollbarX,
          scrollbarXEnabled,
          onScrollbarXEnabledChange: setScrollbarXEnabled,
          scrollbarY,
          onScrollbarYChange: setScrollbarY,
          scrollbarYEnabled,
          onScrollbarYEnabledChange: setScrollbarYEnabled,
          onCornerWidthChange: setCornerWidth,
          onCornerHeightChange: setCornerHeight,
          children: /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
            Primitive.div,
            {
              dir: direction,
              ...scrollAreaProps,
              ref: composedRefs,
              style: {
                position: "relative",
                // Pass corner sizes as CSS vars to reduce re-renders of context consumers
                ["--radix-scroll-area-corner-width"]: cornerWidth + "px",
                ["--radix-scroll-area-corner-height"]: cornerHeight + "px",
                ...props.style
              }
            }
          )
        }
      );
    }
  );
  ScrollArea.displayName = SCROLL_AREA_NAME;
  var VIEWPORT_NAME = "ScrollAreaViewport";
  var ScrollAreaViewport = React211.forwardRef(
    (props, forwardedRef) => {
      const { __scopeScrollArea, children, nonce, ...viewportProps } = props;
      const context = useScrollAreaContext(VIEWPORT_NAME, __scopeScrollArea);
      const ref = React211.useRef(null);
      const composedRefs = useComposedRefs(forwardedRef, ref, context.onViewportChange);
      return /* @__PURE__ */ (0, import_jsx_runtime22.jsxs)(import_jsx_runtime22.Fragment, { children: [
        /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
          "style",
          {
            dangerouslySetInnerHTML: {
              __html: `[data-radix-scroll-area-viewport]{scrollbar-width:none;-ms-overflow-style:none;-webkit-overflow-scrolling:touch;}[data-radix-scroll-area-viewport]::-webkit-scrollbar{display:none}`
            },
            nonce
          }
        ),
        /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
          Primitive.div,
          {
            "data-radix-scroll-area-viewport": "",
            ...viewportProps,
            ref: composedRefs,
            style: {
              /**
               * We don't support `visible` because the intention is to have at least one scrollbar
               * if this component is used and `visible` will behave like `auto` in that case
               * https://developer.mozilla.org/en-US/docs/Web/CSS/overflow#description
               *
               * We don't handle `auto` because the intention is for the native implementation
               * to be hidden if using this component. We just want to ensure the node is scrollable
               * so could have used either `scroll` or `auto` here. We picked `scroll` to prevent
               * the browser from having to work out whether to render native scrollbars or not,
               * we tell it to with the intention of hiding them in CSS.
               */
              overflowX: context.scrollbarXEnabled ? "scroll" : "hidden",
              overflowY: context.scrollbarYEnabled ? "scroll" : "hidden",
              ...props.style
            },
            children: /* @__PURE__ */ (0, import_jsx_runtime22.jsx)("div", { ref: context.onContentChange, style: { minWidth: "100%", display: "table" }, children })
          }
        )
      ] });
    }
  );
  ScrollAreaViewport.displayName = VIEWPORT_NAME;
  var SCROLLBAR_NAME = "ScrollAreaScrollbar";
  var ScrollAreaScrollbar = React211.forwardRef(
    (props, forwardedRef) => {
      const { forceMount, ...scrollbarProps } = props;
      const context = useScrollAreaContext(SCROLLBAR_NAME, props.__scopeScrollArea);
      const { onScrollbarXEnabledChange, onScrollbarYEnabledChange } = context;
      const isHorizontal = props.orientation === "horizontal";
      React211.useEffect(() => {
        isHorizontal ? onScrollbarXEnabledChange(true) : onScrollbarYEnabledChange(true);
        return () => {
          isHorizontal ? onScrollbarXEnabledChange(false) : onScrollbarYEnabledChange(false);
        };
      }, [isHorizontal, onScrollbarXEnabledChange, onScrollbarYEnabledChange]);
      return context.type === "hover" ? /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(ScrollAreaScrollbarHover, { ...scrollbarProps, ref: forwardedRef, forceMount }) : context.type === "scroll" ? /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(ScrollAreaScrollbarScroll, { ...scrollbarProps, ref: forwardedRef, forceMount }) : context.type === "auto" ? /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(ScrollAreaScrollbarAuto, { ...scrollbarProps, ref: forwardedRef, forceMount }) : context.type === "always" ? /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(ScrollAreaScrollbarVisible, { ...scrollbarProps, ref: forwardedRef }) : null;
    }
  );
  ScrollAreaScrollbar.displayName = SCROLLBAR_NAME;
  var ScrollAreaScrollbarHover = React211.forwardRef((props, forwardedRef) => {
    const { forceMount, ...scrollbarProps } = props;
    const context = useScrollAreaContext(SCROLLBAR_NAME, props.__scopeScrollArea);
    const [visible, setVisible] = React211.useState(false);
    React211.useEffect(() => {
      const scrollArea = context.scrollArea;
      let hideTimer = 0;
      if (scrollArea) {
        const handlePointerEnter = () => {
          window.clearTimeout(hideTimer);
          setVisible(true);
        };
        const handlePointerLeave = () => {
          hideTimer = window.setTimeout(() => setVisible(false), context.scrollHideDelay);
        };
        scrollArea.addEventListener("pointerenter", handlePointerEnter);
        scrollArea.addEventListener("pointerleave", handlePointerLeave);
        return () => {
          window.clearTimeout(hideTimer);
          scrollArea.removeEventListener("pointerenter", handlePointerEnter);
          scrollArea.removeEventListener("pointerleave", handlePointerLeave);
        };
      }
    }, [context.scrollArea, context.scrollHideDelay]);
    return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(Presence, { present: forceMount || visible, children: /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
      ScrollAreaScrollbarAuto,
      {
        "data-state": visible ? "visible" : "hidden",
        ...scrollbarProps,
        ref: forwardedRef
      }
    ) });
  });
  var ScrollAreaScrollbarScroll = React211.forwardRef((props, forwardedRef) => {
    const { forceMount, ...scrollbarProps } = props;
    const context = useScrollAreaContext(SCROLLBAR_NAME, props.__scopeScrollArea);
    const isHorizontal = props.orientation === "horizontal";
    const debounceScrollEnd = useDebounceCallback(() => send("SCROLL_END"), 100);
    const [state, send] = useStateMachine2("hidden", {
      hidden: {
        SCROLL: "scrolling"
      },
      scrolling: {
        SCROLL_END: "idle",
        POINTER_ENTER: "interacting"
      },
      interacting: {
        SCROLL: "interacting",
        POINTER_LEAVE: "idle"
      },
      idle: {
        HIDE: "hidden",
        SCROLL: "scrolling",
        POINTER_ENTER: "interacting"
      }
    });
    React211.useEffect(() => {
      if (state === "idle") {
        const hideTimer = window.setTimeout(() => send("HIDE"), context.scrollHideDelay);
        return () => window.clearTimeout(hideTimer);
      }
    }, [state, context.scrollHideDelay, send]);
    React211.useEffect(() => {
      const viewport = context.viewport;
      const scrollDirection = isHorizontal ? "scrollLeft" : "scrollTop";
      if (viewport) {
        let prevScrollPos = viewport[scrollDirection];
        const handleScroll2 = () => {
          const scrollPos = viewport[scrollDirection];
          const hasScrollInDirectionChanged = prevScrollPos !== scrollPos;
          if (hasScrollInDirectionChanged) {
            send("SCROLL");
            debounceScrollEnd();
          }
          prevScrollPos = scrollPos;
        };
        viewport.addEventListener("scroll", handleScroll2);
        return () => viewport.removeEventListener("scroll", handleScroll2);
      }
    }, [context.viewport, isHorizontal, send, debounceScrollEnd]);
    return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(Presence, { present: forceMount || state !== "hidden", children: /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
      ScrollAreaScrollbarVisible,
      {
        "data-state": state === "hidden" ? "hidden" : "visible",
        ...scrollbarProps,
        ref: forwardedRef,
        onPointerEnter: composeEventHandlers(props.onPointerEnter, () => send("POINTER_ENTER")),
        onPointerLeave: composeEventHandlers(props.onPointerLeave, () => send("POINTER_LEAVE"))
      }
    ) });
  });
  var ScrollAreaScrollbarAuto = React211.forwardRef((props, forwardedRef) => {
    const context = useScrollAreaContext(SCROLLBAR_NAME, props.__scopeScrollArea);
    const { forceMount, ...scrollbarProps } = props;
    const [visible, setVisible] = React211.useState(false);
    const isHorizontal = props.orientation === "horizontal";
    const handleResize = useDebounceCallback(() => {
      if (context.viewport) {
        const isOverflowX = context.viewport.offsetWidth < context.viewport.scrollWidth;
        const isOverflowY = context.viewport.offsetHeight < context.viewport.scrollHeight;
        setVisible(isHorizontal ? isOverflowX : isOverflowY);
      }
    }, 10);
    useResizeObserver(context.viewport, handleResize);
    useResizeObserver(context.content, handleResize);
    return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(Presence, { present: forceMount || visible, children: /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
      ScrollAreaScrollbarVisible,
      {
        "data-state": visible ? "visible" : "hidden",
        ...scrollbarProps,
        ref: forwardedRef
      }
    ) });
  });
  var ScrollAreaScrollbarVisible = React211.forwardRef((props, forwardedRef) => {
    const { orientation = "vertical", ...scrollbarProps } = props;
    const context = useScrollAreaContext(SCROLLBAR_NAME, props.__scopeScrollArea);
    const thumbRef = React211.useRef(null);
    const pointerOffsetRef = React211.useRef(0);
    const [sizes, setSizes] = React211.useState({
      content: 0,
      viewport: 0,
      scrollbar: { size: 0, paddingStart: 0, paddingEnd: 0 }
    });
    const thumbRatio = getThumbRatio(sizes.viewport, sizes.content);
    const commonProps = {
      ...scrollbarProps,
      sizes,
      onSizesChange: setSizes,
      hasThumb: Boolean(thumbRatio > 0 && thumbRatio < 1),
      onThumbChange: (thumb) => thumbRef.current = thumb,
      onThumbPointerUp: () => pointerOffsetRef.current = 0,
      onThumbPointerDown: (pointerPos) => pointerOffsetRef.current = pointerPos
    };
    function getScrollPosition(pointerPos, dir) {
      return getScrollPositionFromPointer(pointerPos, pointerOffsetRef.current, sizes, dir);
    }
    if (orientation === "horizontal") {
      return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
        ScrollAreaScrollbarX,
        {
          ...commonProps,
          ref: forwardedRef,
          onThumbPositionChange: () => {
            if (context.viewport && thumbRef.current) {
              const scrollPos = context.viewport.scrollLeft;
              const offset = getThumbOffsetFromScroll(scrollPos, sizes, context.dir);
              thumbRef.current.style.transform = `translate3d(${offset}px, 0, 0)`;
            }
          },
          onWheelScroll: (scrollPos) => {
            if (context.viewport) context.viewport.scrollLeft = scrollPos;
          },
          onDragScroll: (pointerPos) => {
            if (context.viewport) {
              context.viewport.scrollLeft = getScrollPosition(pointerPos, context.dir);
            }
          }
        }
      );
    }
    if (orientation === "vertical") {
      return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
        ScrollAreaScrollbarY,
        {
          ...commonProps,
          ref: forwardedRef,
          onThumbPositionChange: () => {
            if (context.viewport && thumbRef.current) {
              const scrollPos = context.viewport.scrollTop;
              const offset = getThumbOffsetFromScroll(scrollPos, sizes);
              thumbRef.current.style.transform = `translate3d(0, ${offset}px, 0)`;
            }
          },
          onWheelScroll: (scrollPos) => {
            if (context.viewport) context.viewport.scrollTop = scrollPos;
          },
          onDragScroll: (pointerPos) => {
            if (context.viewport) context.viewport.scrollTop = getScrollPosition(pointerPos);
          }
        }
      );
    }
    return null;
  });
  var ScrollAreaScrollbarX = React211.forwardRef((props, forwardedRef) => {
    const { sizes, onSizesChange, ...scrollbarProps } = props;
    const context = useScrollAreaContext(SCROLLBAR_NAME, props.__scopeScrollArea);
    const [computedStyle, setComputedStyle] = React211.useState();
    const ref = React211.useRef(null);
    const composeRefs2 = useComposedRefs(forwardedRef, ref, context.onScrollbarXChange);
    React211.useEffect(() => {
      if (ref.current) setComputedStyle(getComputedStyle(ref.current));
    }, [ref]);
    return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
      ScrollAreaScrollbarImpl,
      {
        "data-orientation": "horizontal",
        ...scrollbarProps,
        ref: composeRefs2,
        sizes,
        style: {
          bottom: 0,
          left: context.dir === "rtl" ? "var(--radix-scroll-area-corner-width)" : 0,
          right: context.dir === "ltr" ? "var(--radix-scroll-area-corner-width)" : 0,
          ["--radix-scroll-area-thumb-width"]: getThumbSize(sizes) + "px",
          ...props.style
        },
        onThumbPointerDown: (pointerPos) => props.onThumbPointerDown(pointerPos.x),
        onDragScroll: (pointerPos) => props.onDragScroll(pointerPos.x),
        onWheelScroll: (event, maxScrollPos) => {
          if (context.viewport) {
            const scrollPos = context.viewport.scrollLeft + event.deltaX;
            props.onWheelScroll(scrollPos);
            if (isScrollingWithinScrollbarBounds(scrollPos, maxScrollPos)) {
              event.preventDefault();
            }
          }
        },
        onResize: () => {
          if (ref.current && context.viewport && computedStyle) {
            onSizesChange({
              content: context.viewport.scrollWidth,
              viewport: context.viewport.offsetWidth,
              scrollbar: {
                size: ref.current.clientWidth,
                paddingStart: toInt(computedStyle.paddingLeft),
                paddingEnd: toInt(computedStyle.paddingRight)
              }
            });
          }
        }
      }
    );
  });
  var ScrollAreaScrollbarY = React211.forwardRef((props, forwardedRef) => {
    const { sizes, onSizesChange, ...scrollbarProps } = props;
    const context = useScrollAreaContext(SCROLLBAR_NAME, props.__scopeScrollArea);
    const [computedStyle, setComputedStyle] = React211.useState();
    const ref = React211.useRef(null);
    const composeRefs2 = useComposedRefs(forwardedRef, ref, context.onScrollbarYChange);
    React211.useEffect(() => {
      if (ref.current) setComputedStyle(getComputedStyle(ref.current));
    }, [ref]);
    return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
      ScrollAreaScrollbarImpl,
      {
        "data-orientation": "vertical",
        ...scrollbarProps,
        ref: composeRefs2,
        sizes,
        style: {
          top: 0,
          right: context.dir === "ltr" ? 0 : void 0,
          left: context.dir === "rtl" ? 0 : void 0,
          bottom: "var(--radix-scroll-area-corner-height)",
          ["--radix-scroll-area-thumb-height"]: getThumbSize(sizes) + "px",
          ...props.style
        },
        onThumbPointerDown: (pointerPos) => props.onThumbPointerDown(pointerPos.y),
        onDragScroll: (pointerPos) => props.onDragScroll(pointerPos.y),
        onWheelScroll: (event, maxScrollPos) => {
          if (context.viewport) {
            const scrollPos = context.viewport.scrollTop + event.deltaY;
            props.onWheelScroll(scrollPos);
            if (isScrollingWithinScrollbarBounds(scrollPos, maxScrollPos)) {
              event.preventDefault();
            }
          }
        },
        onResize: () => {
          if (ref.current && context.viewport && computedStyle) {
            onSizesChange({
              content: context.viewport.scrollHeight,
              viewport: context.viewport.offsetHeight,
              scrollbar: {
                size: ref.current.clientHeight,
                paddingStart: toInt(computedStyle.paddingTop),
                paddingEnd: toInt(computedStyle.paddingBottom)
              }
            });
          }
        }
      }
    );
  });
  var [ScrollbarProvider, useScrollbarContext] = createScrollAreaContext(SCROLLBAR_NAME);
  var ScrollAreaScrollbarImpl = React211.forwardRef((props, forwardedRef) => {
    const {
      __scopeScrollArea,
      sizes,
      hasThumb,
      onThumbChange,
      onThumbPointerUp,
      onThumbPointerDown,
      onThumbPositionChange,
      onDragScroll,
      onWheelScroll,
      onResize,
      ...scrollbarProps
    } = props;
    const context = useScrollAreaContext(SCROLLBAR_NAME, __scopeScrollArea);
    const [scrollbar, setScrollbar] = React211.useState(null);
    const composeRefs2 = useComposedRefs(forwardedRef, (node) => setScrollbar(node));
    const rectRef = React211.useRef(null);
    const prevWebkitUserSelectRef = React211.useRef("");
    const viewport = context.viewport;
    const maxScrollPos = sizes.content - sizes.viewport;
    const handleWheelScroll = useCallbackRef(onWheelScroll);
    const handleThumbPositionChange = useCallbackRef(onThumbPositionChange);
    const handleResize = useDebounceCallback(onResize, 10);
    function handleDragScroll(event) {
      if (rectRef.current) {
        const x = event.clientX - rectRef.current.left;
        const y = event.clientY - rectRef.current.top;
        onDragScroll({ x, y });
      }
    }
    React211.useEffect(() => {
      const handleWheel = (event) => {
        const element = event.target;
        const isScrollbarWheel = scrollbar?.contains(element);
        if (isScrollbarWheel) handleWheelScroll(event, maxScrollPos);
      };
      document.addEventListener("wheel", handleWheel, { passive: false });
      return () => document.removeEventListener("wheel", handleWheel, { passive: false });
    }, [viewport, scrollbar, maxScrollPos, handleWheelScroll]);
    React211.useEffect(handleThumbPositionChange, [sizes, handleThumbPositionChange]);
    useResizeObserver(scrollbar, handleResize);
    useResizeObserver(context.content, handleResize);
    return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
      ScrollbarProvider,
      {
        scope: __scopeScrollArea,
        scrollbar,
        hasThumb,
        onThumbChange: useCallbackRef(onThumbChange),
        onThumbPointerUp: useCallbackRef(onThumbPointerUp),
        onThumbPositionChange: handleThumbPositionChange,
        onThumbPointerDown: useCallbackRef(onThumbPointerDown),
        children: /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
          Primitive.div,
          {
            ...scrollbarProps,
            ref: composeRefs2,
            style: { position: "absolute", ...scrollbarProps.style },
            onPointerDown: composeEventHandlers(props.onPointerDown, (event) => {
              const mainPointer = 0;
              if (event.button === mainPointer) {
                const element = event.target;
                element.setPointerCapture(event.pointerId);
                rectRef.current = scrollbar.getBoundingClientRect();
                prevWebkitUserSelectRef.current = document.body.style.webkitUserSelect;
                document.body.style.webkitUserSelect = "none";
                if (context.viewport) context.viewport.style.scrollBehavior = "auto";
                handleDragScroll(event);
              }
            }),
            onPointerMove: composeEventHandlers(props.onPointerMove, handleDragScroll),
            onPointerUp: composeEventHandlers(props.onPointerUp, (event) => {
              const element = event.target;
              if (element.hasPointerCapture(event.pointerId)) {
                element.releasePointerCapture(event.pointerId);
              }
              document.body.style.webkitUserSelect = prevWebkitUserSelectRef.current;
              if (context.viewport) context.viewport.style.scrollBehavior = "";
              rectRef.current = null;
            })
          }
        )
      }
    );
  });
  var THUMB_NAME = "ScrollAreaThumb";
  var ScrollAreaThumb = React211.forwardRef(
    (props, forwardedRef) => {
      const { forceMount, ...thumbProps } = props;
      const scrollbarContext = useScrollbarContext(THUMB_NAME, props.__scopeScrollArea);
      return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(Presence, { present: forceMount || scrollbarContext.hasThumb, children: /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(ScrollAreaThumbImpl, { ref: forwardedRef, ...thumbProps }) });
    }
  );
  var ScrollAreaThumbImpl = React211.forwardRef(
    (props, forwardedRef) => {
      const { __scopeScrollArea, style, ...thumbProps } = props;
      const scrollAreaContext = useScrollAreaContext(THUMB_NAME, __scopeScrollArea);
      const scrollbarContext = useScrollbarContext(THUMB_NAME, __scopeScrollArea);
      const { onThumbPositionChange } = scrollbarContext;
      const composedRef = useComposedRefs(
        forwardedRef,
        (node) => scrollbarContext.onThumbChange(node)
      );
      const removeUnlinkedScrollListenerRef = React211.useRef(void 0);
      const debounceScrollEnd = useDebounceCallback(() => {
        if (removeUnlinkedScrollListenerRef.current) {
          removeUnlinkedScrollListenerRef.current();
          removeUnlinkedScrollListenerRef.current = void 0;
        }
      }, 100);
      React211.useEffect(() => {
        const viewport = scrollAreaContext.viewport;
        if (viewport) {
          const handleScroll2 = () => {
            debounceScrollEnd();
            if (!removeUnlinkedScrollListenerRef.current) {
              const listener = addUnlinkedScrollListener(viewport, onThumbPositionChange);
              removeUnlinkedScrollListenerRef.current = listener;
              onThumbPositionChange();
            }
          };
          onThumbPositionChange();
          viewport.addEventListener("scroll", handleScroll2);
          return () => viewport.removeEventListener("scroll", handleScroll2);
        }
      }, [scrollAreaContext.viewport, debounceScrollEnd, onThumbPositionChange]);
      return /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
        Primitive.div,
        {
          "data-state": scrollbarContext.hasThumb ? "visible" : "hidden",
          ...thumbProps,
          ref: composedRef,
          style: {
            width: "var(--radix-scroll-area-thumb-width)",
            height: "var(--radix-scroll-area-thumb-height)",
            ...style
          },
          onPointerDownCapture: composeEventHandlers(props.onPointerDownCapture, (event) => {
            const thumb = event.target;
            const thumbRect = thumb.getBoundingClientRect();
            const x = event.clientX - thumbRect.left;
            const y = event.clientY - thumbRect.top;
            scrollbarContext.onThumbPointerDown({ x, y });
          }),
          onPointerUp: composeEventHandlers(props.onPointerUp, scrollbarContext.onThumbPointerUp)
        }
      );
    }
  );
  ScrollAreaThumb.displayName = THUMB_NAME;
  var CORNER_NAME = "ScrollAreaCorner";
  var ScrollAreaCorner = React211.forwardRef(
    (props, forwardedRef) => {
      const context = useScrollAreaContext(CORNER_NAME, props.__scopeScrollArea);
      const hasBothScrollbarsVisible = Boolean(context.scrollbarX && context.scrollbarY);
      const hasCorner = context.type !== "scroll" && hasBothScrollbarsVisible;
      return hasCorner ? /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(ScrollAreaCornerImpl, { ...props, ref: forwardedRef }) : null;
    }
  );
  ScrollAreaCorner.displayName = CORNER_NAME;
  var ScrollAreaCornerImpl = React211.forwardRef((props, forwardedRef) => {
    const { __scopeScrollArea, ...cornerProps } = props;
    const context = useScrollAreaContext(CORNER_NAME, __scopeScrollArea);
    const [width, setWidth] = React211.useState(0);
    const [height, setHeight] = React211.useState(0);
    const hasSize = Boolean(width && height);
    useResizeObserver(context.scrollbarX, () => {
      const height2 = context.scrollbarX?.offsetHeight || 0;
      context.onCornerHeightChange(height2);
      setHeight(height2);
    });
    useResizeObserver(context.scrollbarY, () => {
      const width2 = context.scrollbarY?.offsetWidth || 0;
      context.onCornerWidthChange(width2);
      setWidth(width2);
    });
    return hasSize ? /* @__PURE__ */ (0, import_jsx_runtime22.jsx)(
      Primitive.div,
      {
        ...cornerProps,
        ref: forwardedRef,
        style: {
          width,
          height,
          position: "absolute",
          right: context.dir === "ltr" ? 0 : void 0,
          left: context.dir === "rtl" ? 0 : void 0,
          bottom: 0,
          ...props.style
        }
      }
    ) : null;
  });
  function toInt(value) {
    return value ? parseInt(value, 10) : 0;
  }
  function getThumbRatio(viewportSize, contentSize) {
    const ratio = viewportSize / contentSize;
    return isNaN(ratio) ? 0 : ratio;
  }
  function getThumbSize(sizes) {
    const ratio = getThumbRatio(sizes.viewport, sizes.content);
    const scrollbarPadding = sizes.scrollbar.paddingStart + sizes.scrollbar.paddingEnd;
    const thumbSize = (sizes.scrollbar.size - scrollbarPadding) * ratio;
    return Math.max(thumbSize, 18);
  }
  function getScrollPositionFromPointer(pointerPos, pointerOffset, sizes, dir = "ltr") {
    const thumbSizePx = getThumbSize(sizes);
    const thumbCenter = thumbSizePx / 2;
    const offset = pointerOffset || thumbCenter;
    const thumbOffsetFromEnd = thumbSizePx - offset;
    const minPointerPos = sizes.scrollbar.paddingStart + offset;
    const maxPointerPos = sizes.scrollbar.size - sizes.scrollbar.paddingEnd - thumbOffsetFromEnd;
    const maxScrollPos = sizes.content - sizes.viewport;
    const scrollRange = dir === "ltr" ? [0, maxScrollPos] : [maxScrollPos * -1, 0];
    const interpolate = linearScale([minPointerPos, maxPointerPos], scrollRange);
    return interpolate(pointerPos);
  }
  function getThumbOffsetFromScroll(scrollPos, sizes, dir = "ltr") {
    const thumbSizePx = getThumbSize(sizes);
    const scrollbarPadding = sizes.scrollbar.paddingStart + sizes.scrollbar.paddingEnd;
    const scrollbar = sizes.scrollbar.size - scrollbarPadding;
    const maxScrollPos = sizes.content - sizes.viewport;
    const maxThumbPos = scrollbar - thumbSizePx;
    const scrollClampRange = dir === "ltr" ? [0, maxScrollPos] : [maxScrollPos * -1, 0];
    const scrollWithoutMomentum = clamp(scrollPos, scrollClampRange);
    const interpolate = linearScale([0, maxScrollPos], [0, maxThumbPos]);
    return interpolate(scrollWithoutMomentum);
  }
  function linearScale(input, output) {
    return (value) => {
      if (input[0] === input[1] || output[0] === output[1]) return output[0];
      const ratio = (output[1] - output[0]) / (input[1] - input[0]);
      return output[0] + ratio * (value - input[0]);
    };
  }
  function isScrollingWithinScrollbarBounds(scrollPos, maxScrollPos) {
    return scrollPos > 0 && scrollPos < maxScrollPos;
  }
  var addUnlinkedScrollListener = (node, handler = () => {
  }) => {
    let prevPosition = { left: node.scrollLeft, top: node.scrollTop };
    let rAF = 0;
    (function loop() {
      const position = { left: node.scrollLeft, top: node.scrollTop };
      const isHorizontalScroll = prevPosition.left !== position.left;
      const isVerticalScroll = prevPosition.top !== position.top;
      if (isHorizontalScroll || isVerticalScroll) handler();
      prevPosition = position;
      rAF = window.requestAnimationFrame(loop);
    })();
    return () => window.cancelAnimationFrame(rAF);
  };
  function useDebounceCallback(callback, delay) {
    const handleCallback = useCallbackRef(callback);
    const debounceTimerRef = React211.useRef(0);
    React211.useEffect(() => () => window.clearTimeout(debounceTimerRef.current), []);
    return React211.useCallback(() => {
      window.clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = window.setTimeout(handleCallback, delay);
    }, [handleCallback, delay]);
  }
  function useResizeObserver(element, onResize) {
    const handleResize = useCallbackRef(onResize);
    useLayoutEffect2(() => {
      let rAF = 0;
      if (element) {
        const resizeObserver = new ResizeObserver(() => {
          cancelAnimationFrame(rAF);
          rAF = window.requestAnimationFrame(handleResize);
        });
        resizeObserver.observe(element);
        return () => {
          window.cancelAnimationFrame(rAF);
          resizeObserver.unobserve(element);
        };
      }
    }, [element, handleResize]);
  }
  var Root3 = ScrollArea;
  var Viewport = ScrollAreaViewport;
  var Corner = ScrollAreaCorner;

  // src/components/ui/scroll-area.jsx
  var import_jsx_runtime23 = __toESM(require_jsx_runtime(), 1);
  function ScrollArea2({ className, children, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime23.jsxs)(Root3, { className: cn("ui-scroll-area", className), ...props, children: [
      /* @__PURE__ */ (0, import_jsx_runtime23.jsx)(Viewport, { className: "ui-scroll-area__viewport", children }),
      /* @__PURE__ */ (0, import_jsx_runtime23.jsx)(ScrollBar, {}),
      /* @__PURE__ */ (0, import_jsx_runtime23.jsx)(Corner, { className: "ui-scroll-area__corner" })
    ] });
  }
  function ScrollBar({ className, orientation = "vertical", ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime23.jsx)(
      ScrollAreaScrollbar,
      {
        orientation,
        className: cn("ui-scroll-area__scrollbar", `ui-scroll-area__scrollbar--${orientation}`, className),
        ...props,
        children: /* @__PURE__ */ (0, import_jsx_runtime23.jsx)(ScrollAreaThumb, { className: "ui-scroll-area__thumb" })
      }
    );
  }

  // node_modules/@radix-ui/react-tabs/dist/index.mjs
  var React38 = __toESM(require_react(), 1);

  // node_modules/@radix-ui/react-roving-focus/dist/index.mjs
  var React37 = __toESM(require_react(), 1);
  var import_jsx_runtime24 = __toESM(require_jsx_runtime(), 1);
  var ENTRY_FOCUS = "rovingFocusGroup.onEntryFocus";
  var EVENT_OPTIONS2 = { bubbles: false, cancelable: true };
  var GROUP_NAME = "RovingFocusGroup";
  var [Collection2, useCollection2, createCollectionScope2] = createCollection(GROUP_NAME);
  var [createRovingFocusGroupContext, createRovingFocusGroupScope] = createContextScope(
    GROUP_NAME,
    [createCollectionScope2]
  );
  var [RovingFocusProvider, useRovingFocusContext] = createRovingFocusGroupContext(GROUP_NAME);
  var RovingFocusGroup = React37.forwardRef(
    (props, forwardedRef) => {
      return /* @__PURE__ */ (0, import_jsx_runtime24.jsx)(Collection2.Provider, { scope: props.__scopeRovingFocusGroup, children: /* @__PURE__ */ (0, import_jsx_runtime24.jsx)(Collection2.Slot, { scope: props.__scopeRovingFocusGroup, children: /* @__PURE__ */ (0, import_jsx_runtime24.jsx)(RovingFocusGroupImpl, { ...props, ref: forwardedRef }) }) });
    }
  );
  RovingFocusGroup.displayName = GROUP_NAME;
  var RovingFocusGroupImpl = React37.forwardRef((props, forwardedRef) => {
    const {
      __scopeRovingFocusGroup,
      orientation,
      loop = false,
      dir,
      currentTabStopId: currentTabStopIdProp,
      defaultCurrentTabStopId,
      onCurrentTabStopIdChange,
      onEntryFocus,
      preventScrollOnEntryFocus = false,
      ...groupProps
    } = props;
    const ref = React37.useRef(null);
    const composedRefs = useComposedRefs(forwardedRef, ref);
    const direction = useDirection(dir);
    const [currentTabStopId, setCurrentTabStopId] = useControllableState({
      prop: currentTabStopIdProp,
      defaultProp: defaultCurrentTabStopId ?? null,
      onChange: onCurrentTabStopIdChange,
      caller: GROUP_NAME
    });
    const [isTabbingBackOut, setIsTabbingBackOut] = React37.useState(false);
    const handleEntryFocus = useCallbackRef(onEntryFocus);
    const getItems = useCollection2(__scopeRovingFocusGroup);
    const isClickFocusRef = React37.useRef(false);
    const [focusableItemsCount, setFocusableItemsCount] = React37.useState(0);
    React37.useEffect(() => {
      const node = ref.current;
      if (node) {
        node.addEventListener(ENTRY_FOCUS, handleEntryFocus);
        return () => node.removeEventListener(ENTRY_FOCUS, handleEntryFocus);
      }
    }, [handleEntryFocus]);
    return /* @__PURE__ */ (0, import_jsx_runtime24.jsx)(
      RovingFocusProvider,
      {
        scope: __scopeRovingFocusGroup,
        orientation,
        dir: direction,
        loop,
        currentTabStopId,
        onItemFocus: React37.useCallback(
          (tabStopId) => setCurrentTabStopId(tabStopId),
          [setCurrentTabStopId]
        ),
        onItemShiftTab: React37.useCallback(() => setIsTabbingBackOut(true), []),
        onFocusableItemAdd: React37.useCallback(
          () => setFocusableItemsCount((prevCount) => prevCount + 1),
          []
        ),
        onFocusableItemRemove: React37.useCallback(
          () => setFocusableItemsCount((prevCount) => prevCount - 1),
          []
        ),
        children: /* @__PURE__ */ (0, import_jsx_runtime24.jsx)(
          Primitive.div,
          {
            tabIndex: isTabbingBackOut || focusableItemsCount === 0 ? -1 : 0,
            "data-orientation": orientation,
            ...groupProps,
            ref: composedRefs,
            style: { outline: "none", ...props.style },
            onMouseDown: composeEventHandlers(props.onMouseDown, () => {
              isClickFocusRef.current = true;
            }),
            onFocus: composeEventHandlers(props.onFocus, (event) => {
              const isKeyboardFocus = !isClickFocusRef.current;
              if (event.target === event.currentTarget && isKeyboardFocus && !isTabbingBackOut) {
                const entryFocusEvent = new CustomEvent(ENTRY_FOCUS, EVENT_OPTIONS2);
                event.currentTarget.dispatchEvent(entryFocusEvent);
                if (!entryFocusEvent.defaultPrevented) {
                  const items = getItems().filter((item) => item.focusable);
                  const activeItem = items.find((item) => item.active);
                  const currentItem = items.find((item) => item.id === currentTabStopId);
                  const candidateItems = [activeItem, currentItem, ...items].filter(
                    Boolean
                  );
                  const candidateNodes = candidateItems.map((item) => item.ref.current);
                  focusFirst2(candidateNodes, preventScrollOnEntryFocus);
                }
              }
              isClickFocusRef.current = false;
            }),
            onBlur: composeEventHandlers(props.onBlur, () => setIsTabbingBackOut(false))
          }
        )
      }
    );
  });
  var ITEM_NAME2 = "RovingFocusGroupItem";
  var RovingFocusGroupItem = React37.forwardRef(
    (props, forwardedRef) => {
      const {
        __scopeRovingFocusGroup,
        focusable = true,
        active = false,
        tabStopId,
        children,
        ...itemProps
      } = props;
      const autoId = useId();
      const id = tabStopId || autoId;
      const context = useRovingFocusContext(ITEM_NAME2, __scopeRovingFocusGroup);
      const isCurrentTabStop = context.currentTabStopId === id;
      const getItems = useCollection2(__scopeRovingFocusGroup);
      const { onFocusableItemAdd, onFocusableItemRemove, currentTabStopId } = context;
      React37.useEffect(() => {
        if (focusable) {
          onFocusableItemAdd();
          return () => onFocusableItemRemove();
        }
      }, [focusable, onFocusableItemAdd, onFocusableItemRemove]);
      return /* @__PURE__ */ (0, import_jsx_runtime24.jsx)(
        Collection2.ItemSlot,
        {
          scope: __scopeRovingFocusGroup,
          id,
          focusable,
          active,
          children: /* @__PURE__ */ (0, import_jsx_runtime24.jsx)(
            Primitive.span,
            {
              tabIndex: isCurrentTabStop ? 0 : -1,
              "data-orientation": context.orientation,
              ...itemProps,
              ref: forwardedRef,
              onMouseDown: composeEventHandlers(props.onMouseDown, (event) => {
                if (!focusable) event.preventDefault();
                else context.onItemFocus(id);
              }),
              onFocus: composeEventHandlers(props.onFocus, () => context.onItemFocus(id)),
              onKeyDown: composeEventHandlers(props.onKeyDown, (event) => {
                if (event.key === "Tab" && event.shiftKey) {
                  context.onItemShiftTab();
                  return;
                }
                if (event.target !== event.currentTarget) return;
                const focusIntent = getFocusIntent(event, context.orientation, context.dir);
                if (focusIntent !== void 0) {
                  if (event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) return;
                  event.preventDefault();
                  const items = getItems().filter((item) => item.focusable);
                  let candidateNodes = items.map((item) => item.ref.current);
                  if (focusIntent === "last") candidateNodes.reverse();
                  else if (focusIntent === "prev" || focusIntent === "next") {
                    if (focusIntent === "prev") candidateNodes.reverse();
                    const currentIndex = candidateNodes.indexOf(event.currentTarget);
                    candidateNodes = context.loop ? wrapArray(candidateNodes, currentIndex + 1) : candidateNodes.slice(currentIndex + 1);
                  }
                  setTimeout(() => focusFirst2(candidateNodes));
                }
              }),
              children: typeof children === "function" ? children({ isCurrentTabStop, hasTabStop: currentTabStopId != null }) : children
            }
          )
        }
      );
    }
  );
  RovingFocusGroupItem.displayName = ITEM_NAME2;
  var MAP_KEY_TO_FOCUS_INTENT = {
    ArrowLeft: "prev",
    ArrowUp: "prev",
    ArrowRight: "next",
    ArrowDown: "next",
    PageUp: "first",
    Home: "first",
    PageDown: "last",
    End: "last"
  };
  function getDirectionAwareKey(key, dir) {
    if (dir !== "rtl") return key;
    return key === "ArrowLeft" ? "ArrowRight" : key === "ArrowRight" ? "ArrowLeft" : key;
  }
  function getFocusIntent(event, orientation, dir) {
    const key = getDirectionAwareKey(event.key, dir);
    if (orientation === "vertical" && ["ArrowLeft", "ArrowRight"].includes(key)) return void 0;
    if (orientation === "horizontal" && ["ArrowUp", "ArrowDown"].includes(key)) return void 0;
    return MAP_KEY_TO_FOCUS_INTENT[key];
  }
  function focusFirst2(candidates, preventScroll = false) {
    const PREVIOUSLY_FOCUSED_ELEMENT = document.activeElement;
    for (const candidate of candidates) {
      if (candidate === PREVIOUSLY_FOCUSED_ELEMENT) return;
      candidate.focus({ preventScroll });
      if (document.activeElement !== PREVIOUSLY_FOCUSED_ELEMENT) return;
    }
  }
  function wrapArray(array, startIndex) {
    return array.map((_, index) => array[(startIndex + index) % array.length]);
  }
  var Root4 = RovingFocusGroup;
  var Item2 = RovingFocusGroupItem;

  // node_modules/@radix-ui/react-tabs/dist/index.mjs
  var import_jsx_runtime25 = __toESM(require_jsx_runtime(), 1);
  var TABS_NAME = "Tabs";
  var [createTabsContext, createTabsScope] = createContextScope(TABS_NAME, [
    createRovingFocusGroupScope
  ]);
  var useRovingFocusGroupScope = createRovingFocusGroupScope();
  var [TabsProvider, useTabsContext] = createTabsContext(TABS_NAME);
  var Tabs = React38.forwardRef(
    (props, forwardedRef) => {
      const {
        __scopeTabs,
        value: valueProp,
        onValueChange,
        defaultValue,
        orientation = "horizontal",
        dir,
        activationMode = "automatic",
        ...tabsProps
      } = props;
      const direction = useDirection(dir);
      const [value, setValue] = useControllableState({
        prop: valueProp,
        onChange: onValueChange,
        defaultProp: defaultValue ?? "",
        caller: TABS_NAME
      });
      return /* @__PURE__ */ (0, import_jsx_runtime25.jsx)(
        TabsProvider,
        {
          scope: __scopeTabs,
          baseId: useId(),
          value,
          onValueChange: setValue,
          orientation,
          dir: direction,
          activationMode,
          children: /* @__PURE__ */ (0, import_jsx_runtime25.jsx)(
            Primitive.div,
            {
              dir: direction,
              "data-orientation": orientation,
              ...tabsProps,
              ref: forwardedRef
            }
          )
        }
      );
    }
  );
  Tabs.displayName = TABS_NAME;
  var TAB_LIST_NAME = "TabsList";
  var TabsList = React38.forwardRef(
    (props, forwardedRef) => {
      const { __scopeTabs, loop = true, ...listProps } = props;
      const context = useTabsContext(TAB_LIST_NAME, __scopeTabs);
      const rovingFocusGroupScope = useRovingFocusGroupScope(__scopeTabs);
      return /* @__PURE__ */ (0, import_jsx_runtime25.jsx)(
        Root4,
        {
          asChild: true,
          ...rovingFocusGroupScope,
          orientation: context.orientation,
          dir: context.dir,
          loop,
          children: /* @__PURE__ */ (0, import_jsx_runtime25.jsx)(
            Primitive.div,
            {
              role: "tablist",
              "aria-orientation": context.orientation,
              ...listProps,
              ref: forwardedRef
            }
          )
        }
      );
    }
  );
  TabsList.displayName = TAB_LIST_NAME;
  var TRIGGER_NAME4 = "TabsTrigger";
  var TabsTrigger = React38.forwardRef(
    (props, forwardedRef) => {
      const { __scopeTabs, value, disabled = false, ...triggerProps } = props;
      const context = useTabsContext(TRIGGER_NAME4, __scopeTabs);
      const rovingFocusGroupScope = useRovingFocusGroupScope(__scopeTabs);
      const triggerId = makeTriggerId(context.baseId, value);
      const contentId = makeContentId(context.baseId, value);
      const isSelected = value === context.value;
      return /* @__PURE__ */ (0, import_jsx_runtime25.jsx)(
        Item2,
        {
          asChild: true,
          ...rovingFocusGroupScope,
          focusable: !disabled,
          active: isSelected,
          children: /* @__PURE__ */ (0, import_jsx_runtime25.jsx)(
            Primitive.button,
            {
              type: "button",
              role: "tab",
              "aria-selected": isSelected,
              "aria-controls": contentId,
              "data-state": isSelected ? "active" : "inactive",
              "data-disabled": disabled ? "" : void 0,
              disabled,
              id: triggerId,
              ...triggerProps,
              ref: forwardedRef,
              onMouseDown: composeEventHandlers(props.onMouseDown, (event) => {
                if (!disabled && event.button === 0 && event.ctrlKey === false) {
                  context.onValueChange(value);
                } else {
                  event.preventDefault();
                }
              }),
              onKeyDown: composeEventHandlers(props.onKeyDown, (event) => {
                if ([" ", "Enter"].includes(event.key)) context.onValueChange(value);
              }),
              onFocus: composeEventHandlers(props.onFocus, () => {
                const isAutomaticActivation = context.activationMode !== "manual";
                if (!isSelected && !disabled && isAutomaticActivation) {
                  context.onValueChange(value);
                }
              })
            }
          )
        }
      );
    }
  );
  TabsTrigger.displayName = TRIGGER_NAME4;
  var CONTENT_NAME4 = "TabsContent";
  var TabsContent = React38.forwardRef(
    (props, forwardedRef) => {
      const { __scopeTabs, value, forceMount, children, ...contentProps } = props;
      const context = useTabsContext(CONTENT_NAME4, __scopeTabs);
      const triggerId = makeTriggerId(context.baseId, value);
      const contentId = makeContentId(context.baseId, value);
      const isSelected = value === context.value;
      const isMountAnimationPreventedRef = React38.useRef(isSelected);
      React38.useEffect(() => {
        const rAF = requestAnimationFrame(() => isMountAnimationPreventedRef.current = false);
        return () => cancelAnimationFrame(rAF);
      }, []);
      return /* @__PURE__ */ (0, import_jsx_runtime25.jsx)(Presence, { present: forceMount || isSelected, children: ({ present }) => /* @__PURE__ */ (0, import_jsx_runtime25.jsx)(
        Primitive.div,
        {
          "data-state": isSelected ? "active" : "inactive",
          "data-orientation": context.orientation,
          role: "tabpanel",
          "aria-labelledby": triggerId,
          hidden: !present,
          id: contentId,
          tabIndex: 0,
          ...contentProps,
          ref: forwardedRef,
          style: {
            ...props.style,
            animationDuration: isMountAnimationPreventedRef.current ? "0s" : void 0
          },
          children: present && children
        }
      ) });
    }
  );
  TabsContent.displayName = CONTENT_NAME4;
  function makeTriggerId(baseId, value) {
    return `${baseId}-trigger-${value}`;
  }
  function makeContentId(baseId, value) {
    return `${baseId}-content-${value}`;
  }
  var Root23 = Tabs;
  var List = TabsList;
  var Trigger3 = TabsTrigger;

  // src/components/ui/tabs.jsx
  var import_jsx_runtime26 = __toESM(require_jsx_runtime(), 1);
  function Tabs2(props) {
    return /* @__PURE__ */ (0, import_jsx_runtime26.jsx)(Root23, { className: cn("ui-tabs", props.className), ...props });
  }
  function TabsList2({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime26.jsx)(List, { className: cn("ui-tabs__list", className), ...props });
  }
  function TabsTrigger2({ className, ...props }) {
    return /* @__PURE__ */ (0, import_jsx_runtime26.jsx)(Trigger3, { className: cn("ui-tabs__trigger", className), ...props });
  }

  // src/index.jsx
  var import_jsx_runtime27 = __toESM(require_jsx_runtime(), 1);
  var API = "";
  function cx2(...parts) {
    return parts.filter(Boolean).join(" ");
  }
  function RelayMark({ className, title = "Relay" }) {
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { className, viewBox: "0 0 256 256", role: "img", "aria-label": title, children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { fill: "#E8E6E3", d: "M128 138L92 110L82 122L118 150V190H138V150L174 122L164 110Z" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "82", cy: "122", r: "18", fill: "#E8E6E3" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "128", cy: "200", r: "18", fill: "#E8E6E3" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "174", cy: "122", r: "18", fill: "#6FC2A6" })
    ] });
  }
  var NAV_ICONS = {
    overview: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "16", height: "16", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "1.5", strokeLinecap: "round", strokeLinejoin: "round", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("rect", { x: "3", y: "3", width: "7", height: "7" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("rect", { x: "14", y: "3", width: "7", height: "7" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("rect", { x: "3", y: "14", width: "7", height: "7" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("rect", { x: "14", y: "14", width: "7", height: "7" })
    ] }),
    deployments: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "16", height: "16", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "1.5", strokeLinecap: "round", strokeLinejoin: "round", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("line", { x1: "6", y1: "3", x2: "6", y2: "15" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "18", cy: "6", r: "3" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "6", cy: "18", r: "3" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M18 9a9 9 0 0 1-9 9" })
    ] }),
    logs: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "16", height: "16", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "1.5", strokeLinecap: "round", strokeLinejoin: "round", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M4 6h16" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M4 12h10" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M4 18h13" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "18", cy: "12", r: "2.5" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "20", cy: "18", r: "1.5" })
    ] }),
    settings: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "16", height: "16", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "1.5", strokeLinecap: "round", strokeLinejoin: "round", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M20 7H4" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M4 12h16" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M4 17h16" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "8", cy: "7", r: "2" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "16", cy: "12", r: "2" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("circle", { cx: "8", cy: "17", r: "2" })
    ] })
  };
  async function api(path, options = {}) {
    const res = await fetch(`${API}${path}`, {
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        "X-Requested-With": "relay-dashboard",
        ...options.headers || {}
      },
      ...options
    });
    const contentType = res.headers.get("content-type") || "";
    const data = contentType.includes("application/json") ? await res.json() : await res.text();
    if (!res.ok) {
      const error = typeof data === "object" && data?.error ? data.error : `HTTP ${res.status}`;
      const err = new Error(error);
      err.status = res.status;
      throw err;
    }
    return data;
  }
  function deployKey(app, env, branch) {
    return `${app}__${env}__${branch}`;
  }
  function prettyDuration(ms) {
    if (!Number.isFinite(ms) || ms < 0) return "0ms";
    if (ms < 1e3) return `${ms}ms`;
    if (ms < 6e4) return `${(ms / 1e3).toFixed(ms < 1e4 ? 1 : 0)}s`;
    const minutes = Math.floor(ms / 6e4);
    const seconds = Math.round(ms % 6e4 / 1e3);
    return `${minutes}m ${seconds}s`;
  }
  function timeAgo(input) {
    if (!input) return "now";
    const value = new Date(input);
    const diff = Math.max(0, Date.now() - value.getTime());
    const sec = Math.floor(diff / 1e3);
    if (sec < 60) return `${sec}s`;
    const min = Math.floor(sec / 60);
    if (min < 60) return `${min}m`;
    const hr = Math.floor(min / 60);
    if (hr < 24) return `${hr}h`;
    const day = Math.floor(hr / 24);
    if (day < 30) return `${day}d`;
    const month = Math.floor(day / 30);
    if (month < 12) return `${month}mo`;
    return `${Math.floor(month / 12)}y`;
  }
  function computePreviewURL(envInfo, deploy) {
    if (deploy?.preview_url) return deploy.preview_url;
    if (envInfo?.public_host) return `https://${envInfo.public_host}`;
    if (envInfo?.host_port) return `${window.location.protocol}//${window.location.hostname}:${envInfo.host_port}`;
    return "";
  }
  function computeConfiguredURL(envInfo) {
    if (envInfo?.public_host) return `https://${envInfo.public_host}`;
    if ((envInfo?.mode || "port") === "port" && envInfo?.host_port) {
      return `${window.location.protocol}//${window.location.hostname}:${envInfo.host_port}`;
    }
    return "";
  }
  function deployDurationMs(deploy) {
    if (!deploy?.started_at || !deploy?.ended_at) return 0;
    return Math.max(0, new Date(deploy.ended_at) - new Date(deploy.started_at));
  }
  function formatPercent(value) {
    if (!Number.isFinite(value)) return "0%";
    return `${Math.round(value)}%`;
  }
  function hasSlotRollout(envInfo) {
    return Boolean(envInfo?.active_slot);
  }
  function rolloutStrategy(envInfo) {
    return hasSlotRollout(envInfo) ? "new / old handoff" : "single target";
  }
  function liveTargetLabel(envInfo) {
    return hasSlotRollout(envInfo) ? "new live" : "single target";
  }
  function oldTargetLabel(envInfo) {
    if (!hasSlotRollout(envInfo)) return "not staged";
    return envInfo?.standby_slot ? "old draining" : "removed";
  }
  function drainWindowLabel(envInfo) {
    if (!hasSlotRollout(envInfo)) return "n/a";
    if (!envInfo?.standby_slot) return "complete";
    const remaining = Number(envInfo?.drain_until || 0) - Date.now();
    return remaining > 0 ? prettyDuration(remaining) : "expiring";
  }
  function internalSlotLabel(slot) {
    return slot ? `Internal slot id: ${slot}.` : "";
  }
  function targetInspectURL(baseURL, target) {
    if (!baseURL) return "";
    try {
      const url = new URL(baseURL, window.location.href);
      url.searchParams.set("__relay_target", target);
      return url.toString();
    } catch {
      return "";
    }
  }
  function trafficModeLabel(value) {
    return value === "session" ? "session sticky" : "edge cutover";
  }
  function engineLabel(value) {
    return value === "vessel" ? "Vessel" : "Docker";
  }
  function normalizeEngineValue(value) {
    return value === "vessel" ? "vessel" : "docker";
  }
  function applyEngineConstraints(config) {
    return {
      ...config,
      engine: normalizeEngineValue(config?.engine)
    };
  }
  function buildSettingsConfig(selectedEnv) {
    return applyEngineConstraints({
      repo_url: selectedEnv?.repo_url || "",
      engine: normalizeEngineValue(selectedEnv?.engine),
      mode: selectedEnv?.mode || "port",
      traffic_mode: selectedEnv?.traffic_mode || "edge",
      host_port: selectedEnv?.host_port || 0,
      service_port: selectedEnv?.service_port || 0,
      public_host: selectedEnv?.public_host || "",
      webhook_secret: selectedEnv?.webhook_secret || ""
    });
  }
  function prettyCompanionType(value) {
    const kind = (value || "custom").toLowerCase();
    if (kind === "postgres") return "Postgres";
    if (kind === "redis") return "Redis";
    if (kind === "mysql") return "MySQL";
    if (kind === "mongo") return "Mongo";
    if (kind === "worker") return "Worker";
    return "Custom";
  }
  function defaultCompanionDraft(kind = "postgres") {
    const base = {
      name: "",
      type: kind,
      version: "",
      image: "",
      command: "",
      stopped: false,
      port: 0,
      host_port: 0,
      env: {},
      volumes: [],
      health: {
        test: "",
        interval_seconds: 0,
        timeout_seconds: 0,
        retries: 0,
        start_period_seconds: 0
      }
    };
    if (kind === "postgres") {
      return { ...base, name: "db", version: "16", port: 5432 };
    }
    if (kind === "redis") {
      return { ...base, name: "cache", version: "7", port: 6379 };
    }
    if (kind === "worker") {
      return { ...base, name: "worker", image: "ghcr.io/your-org/worker:latest", command: "node worker.js" };
    }
    return { ...base, name: "custom", type: "custom", image: "ghcr.io/your-org/service:latest" };
  }
  function envToText(env) {
    return Object.entries(env || {}).map(([key, value]) => `${key}=${value}`).join("\n");
  }
  function textToEnv(text) {
    return text.split(/\r?\n/).map((line) => line.trim()).filter(Boolean).reduce((acc, line) => {
      const idx = line.indexOf("=");
      if (idx <= 0) return acc;
      acc[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
      return acc;
    }, {});
  }
  function normalizeProjects(projects) {
    return (projects || []).map((project) => ({
      ...project,
      envs: (project?.envs || []).map((env) => ({
        ...env,
        engine: env?.engine || "docker"
      })),
      services: project?.services || []
    }));
  }
  var ACTIVE_STATUSES = /* @__PURE__ */ new Set(["queued", "running", "building"]);
  function hasActiveDeploysIn(deploys) {
    return deploys.some((d) => ACTIVE_STATUSES.has(d.status));
  }
  function deployStatusClass(status) {
    if (status === "success") return "status-chip--ok";
    if (status === "failed" || status === "error") return "status-chip--danger";
    return "status-chip--warn";
  }
  function operationLabel(source) {
    if (source === "rollback") return "rollback";
    if (source === "git") return "git deploy";
    if (source === "sync") return "sync deploy";
    return "deploy";
  }
  function operationClass(source) {
    if (source === "rollback") return "operation-chip--rollback";
    if (source === "git") return "operation-chip--git";
    if (source === "sync") return "operation-chip--sync";
    return "";
  }
  function deployPhaseText(deploy) {
    if (!deploy) return "idle";
    if (deploy.source === "rollback" && ACTIVE_STATUSES.has(deploy.status)) return "rollback in progress";
    if (deploy.source === "rollback" && deploy.status === "success") return "rollback complete";
    if (deploy.source === "rollback" && (deploy.status === "failed" || deploy.status === "error")) return "rollback failed";
    if (ACTIVE_STATUSES.has(deploy.status)) return "deploy in progress";
    return deploy.status;
  }
  function shortImageTag(image) {
    if (!image) return "n/a";
    const last = image.split(":").pop() || image;
    return last.length > 14 ? last.slice(0, 14) : last;
  }
  function deployDurationLabel(deploy) {
    const duration = deployDurationMs(deploy);
    if (duration) return prettyDuration(duration);
    return ACTIVE_STATUSES.has(deploy?.status) ? "running" : "n/a";
  }
  function formatDateTime(input) {
    if (!input) return "Waiting";
    const value = new Date(input);
    if (!Number.isFinite(value.getTime())) return "Unknown";
    return new Intl.DateTimeFormat(void 0, {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit"
    }).format(value);
  }
  function formatCommitSHA(value) {
    return value ? value.slice(0, 7) : "manual";
  }
  function sanitizeRepoURL(value) {
    if (!value) return "";
    return value.replace(/^https?:\/\//i, "").replace(/^git@/i, "").replace(":", "/").replace(/\.git$/i, "").replace(/\/$/, "");
  }
  function projectRepoURL(project) {
    return (project?.envs || []).map((envInfo) => envInfo.repo_url).find(Boolean) || "";
  }
  function repoProviderInfo(repoURL) {
    if (!repoURL) {
      return {
        connected: false,
        vendor: "Manual",
        label: "Manual deploy only",
        host: "No Git remote linked yet.",
        tone: "muted"
      };
    }
    const host = sanitizeRepoURL(repoURL);
    const lower = host.toLowerCase();
    if (lower.includes("github.com")) return { connected: true, vendor: "GitHub", label: "GitHub connected", host, tone: "teal" };
    if (lower.includes("gitlab.com")) return { connected: true, vendor: "GitLab", label: "GitLab connected", host, tone: "amber" };
    if (lower.includes("bitbucket")) return { connected: true, vendor: "Bitbucket", label: "Bitbucket connected", host, tone: "teal" };
    return { connected: true, vendor: "Git", label: "Git remote linked", host, tone: "amber" };
  }
  function logLineTone(line) {
    const lower = (line || "").toLowerCase();
    if (!lower) return "muted";
    if (lower.includes("error") || lower.includes("failed") || lower.includes("panic") || lower.includes("fatal")) return "danger";
    if (lower.includes("warn")) return "warn";
    if (lower.includes("ready") || lower.includes("done") || lower.includes("complete") || lower.includes("success")) return "ok";
    return "neutral";
  }
  function buildLogStats(lines) {
    return lines.reduce(
      (acc, line) => {
        acc.total += 1;
        const tone = logLineTone(line);
        if (tone === "danger") acc.errors += 1;
        if (tone === "warn") acc.warnings += 1;
        if (tone === "ok") acc.success += 1;
        return acc;
      },
      { total: 0, errors: 0, warnings: 0, success: 0 }
    );
  }
  function runtimeLogLevel(line) {
    const lower = (line || "").toLowerCase();
    if (lower.includes("fatal") || lower.includes("panic")) return "fatal";
    if (lower.includes("error") || lower.includes("failed")) return "error";
    if (lower.includes("warn")) return "warning";
    return "info";
  }
  function runtimeLogLevelVariant(level) {
    if (level === "fatal" || level === "error") return "danger";
    if (level === "warning") return "warning";
    return "muted";
  }
  function parseRuntimeLogEntry(raw, targetLabel) {
    const match = String(raw || "").match(/^(\d{4}-\d{2}-\d{2}T[^\s]+)\s(.*)$/);
    const timestamp = match?.[1] || "";
    const message = match?.[2] || String(raw || "");
    const level = runtimeLogLevel(message);
    return {
      raw: String(raw || ""),
      timestamp,
      time: timestamp ? formatDateTime(timestamp) : formatDateTime((/* @__PURE__ */ new Date()).toISOString()),
      message,
      level,
      host: targetLabel || "runtime",
      request: targetLabel || "runtime"
    };
  }
  function sinceISO(windowFilter) {
    const map = {
      "30m": 30 * 60 * 1e3,
      "6h": 6 * 60 * 60 * 1e3,
      "24h": 24 * 60 * 60 * 1e3
    };
    return new Date(Date.now() - (map[windowFilter] || map["30m"])).toISOString();
  }
  function runtimeFilterMatches(entry, levelFilter, query) {
    const matchesLevel = levelFilter === "all" || levelFilter === "warning" && entry.level === "warning" || levelFilter === "error" && entry.level === "error" || levelFilter === "fatal" && entry.level === "fatal";
    if (!matchesLevel) return false;
    const normalizedQuery = query.trim().toLowerCase();
    if (!normalizedQuery) return true;
    return [entry.message, entry.host, entry.request, entry.level].filter(Boolean).some((value) => String(value).toLowerCase().includes(normalizedQuery));
  }
  var EMPTY_RUNTIME_LANE_STATE = Object.freeze({
    appStopped: false,
    appRunning: false,
    hasRunningTarget: false,
    activeSlot: "",
    standbySlot: "",
    offlineReason: ""
  });
  function normalizeRuntimeLaneState(lane, selectedEnv) {
    return {
      appStopped: Boolean(lane?.app_stopped ?? selectedEnv?.stopped),
      appRunning: Boolean(lane?.app_running),
      hasRunningTarget: Boolean(lane?.has_running_target),
      activeSlot: lane?.active_slot || "",
      standbySlot: lane?.standby_slot || "",
      offlineReason: lane?.offline_reason || (selectedEnv?.stopped ? "This app lane is currently off. Start or redeploy it to resume runtime logs." : "")
    };
  }
  function isRuntimeOfflineError(message, laneState) {
    const normalized = String(message || "").trim().toLowerCase();
    if (!normalized) return Boolean(laneState?.appStopped);
    return Boolean(laneState?.appStopped) || normalized.includes("no such container") || normalized.includes("currently off") || normalized.includes("offline");
  }
  function humanizeRuntimeStreamError(message, laneState, targetMeta) {
    const raw = String(message || "").trim();
    if (!raw) return laneState?.offlineReason || "";
    if (/no such container/i.test(raw) || /exit status 1/i.test(raw) && laneState?.offlineReason) {
      return laneState?.offlineReason || `${targetMeta?.label || "Selected runtime target"} is offline right now.`;
    }
    return raw;
  }
  function describeRuntimeLaneState(selectedEnv, laneState, runningTargetCount) {
    const liveLabel = `${runningTargetCount} live target${runningTargetCount === 1 ? "" : "s"}`;
    if (laneState.appStopped || selectedEnv?.stopped) {
      return {
        title: "App lane is off",
        body: `${laneState.offlineReason || "This app lane is currently off. Start or redeploy it to resume runtime logs."}${runningTargetCount ? " Running helper targets can still be inspected below." : ""}`,
        badgeLabel: runningTargetCount ? liveLabel : "No live targets",
        badgeVariant: "warning"
      };
    }
    if (!laneState.appRunning) {
      return {
        title: "Live app container unavailable",
        body: laneState.offlineReason || "Relay cannot find a running app container for this lane yet.",
        badgeLabel: runningTargetCount ? liveLabel : "No live targets",
        badgeVariant: "warning"
      };
    }
    if (!laneState.hasRunningTarget) {
      return {
        title: "No runtime targets online",
        body: laneState.offlineReason || "Runtime containers for this lane are not currently running.",
        badgeLabel: "No live targets",
        badgeVariant: "muted"
      };
    }
    return {
      title: "Runtime streaming ready",
      body: "Select a live target to follow container output in real time.",
      badgeLabel: liveLabel,
      badgeVariant: "success"
    };
  }
  function useDashboardData(enabled) {
    const [state, setState] = (0, import_react9.useState)({
      loading: true,
      refreshing: false,
      projects: [],
      deploys: [],
      error: ""
    });
    function applyData(projects, deploys) {
      const sorted = (deploys || []).slice().sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
      setState({ loading: false, refreshing: false, projects: normalizeProjects(projects), deploys: sorted, error: "" });
    }
    (0, import_react9.useEffect)(() => {
      if (!enabled) return void 0;
      const es = new EventSource(`${API}/api/events`, { withCredentials: true });
      es.addEventListener("snapshot", (e) => {
        try {
          const { projects, deploys } = JSON.parse(e.data);
          applyData(projects, deploys);
        } catch {
        }
      });
      es.addEventListener("update", (e) => {
        try {
          const { projects, deploys } = JSON.parse(e.data);
          applyData(projects, deploys);
        } catch {
        }
      });
      es.onerror = () => {
        setState((prev) => ({
          ...prev,
          loading: false,
          error: prev.loading ? "Cannot connect to agent" : "Connection lost \u2014 reconnecting\u2026"
        }));
      };
      return () => es.close();
    }, [enabled]);
    const isLive = state.deploys.length > 0 && hasActiveDeploysIn(state.deploys);
    const manualRefresh = (0, import_react9.useCallback)(async () => {
      setState((prev) => ({ ...prev, refreshing: true }));
      try {
        const [projects, deploys] = await Promise.all([api("/api/projects"), api("/api/deploys")]);
        applyData(projects, deploys);
      } catch (err) {
        setState((prev) => ({ ...prev, refreshing: false, error: err.message }));
      }
    }, []);
    return [{ ...state, isLive }, manualRefresh];
  }
  function useRuntimeLogs(selectedEnv, selectedTarget, windowFilter) {
    const [state, setState] = (0, import_react9.useState)({
      loading: false,
      targets: [],
      defaultTarget: "",
      lines: [],
      status: "idle",
      targetMeta: null,
      lane: EMPTY_RUNTIME_LANE_STATE,
      error: ""
    });
    (0, import_react9.useEffect)(() => {
      if (!selectedEnv?.app || !selectedEnv?.env || !selectedEnv?.branch) {
        setState({
          loading: false,
          targets: [],
          defaultTarget: "",
          lines: [],
          status: "idle",
          targetMeta: null,
          lane: EMPTY_RUNTIME_LANE_STATE,
          error: ""
        });
        return void 0;
      }
      let cancelled = false;
      setState((prev) => ({ ...prev, loading: true, error: "" }));
      api(`/api/runtime/logs/targets?app=${encodeURIComponent(selectedEnv.app)}&env=${encodeURIComponent(selectedEnv.env)}&branch=${encodeURIComponent(selectedEnv.branch)}`).then((data) => {
        if (cancelled) return;
        const lane = normalizeRuntimeLaneState(data?.lane, selectedEnv);
        const defaultTarget = data?.default_target || "";
        setState((prev) => ({
          ...prev,
          loading: false,
          targets: data?.targets || [],
          defaultTarget,
          lines: [],
          status: defaultTarget ? "idle" : lane.hasRunningTarget ? "idle" : "offline",
          targetMeta: null,
          lane,
          error: ""
        }));
      }).catch((err) => {
        if (cancelled) return;
        setState((prev) => ({
          ...prev,
          loading: false,
          targets: [],
          defaultTarget: "",
          lines: [],
          status: "error",
          targetMeta: null,
          lane: normalizeRuntimeLaneState(null, selectedEnv),
          error: err.message
        }));
      });
      return () => {
        cancelled = true;
      };
    }, [selectedEnv]);
    (0, import_react9.useEffect)(() => {
      if (!selectedEnv?.app || !selectedEnv?.env || !selectedEnv?.branch) return void 0;
      const effectiveTarget = selectedTarget || state.defaultTarget;
      const effectiveTargetMeta = state.targets.find((target) => target.id === effectiveTarget) || null;
      const laneState = state.lane;
      if (!effectiveTarget) {
        setState((prev) => ({
          ...prev,
          lines: [],
          status: prev.lane.hasRunningTarget ? "idle" : "offline",
          targetMeta: null,
          error: ""
        }));
        return void 0;
      }
      if (effectiveTargetMeta && !effectiveTargetMeta.running) {
        setState((prev) => ({
          ...prev,
          lines: [],
          status: "offline",
          targetMeta: effectiveTargetMeta,
          error: ""
        }));
        return void 0;
      }
      const controller = new AbortController();
      let cancelled = false;
      setState((prev) => ({ ...prev, lines: [], status: "connecting", targetMeta: effectiveTargetMeta, error: "" }));
      async function run() {
        try {
          const params = new URLSearchParams({
            app: selectedEnv.app,
            env: selectedEnv.env,
            branch: selectedEnv.branch,
            target: effectiveTarget,
            tail: "300",
            since: sinceISO(windowFilter)
          });
          const res = await fetch(`/api/runtime/logs/stream?${params.toString()}`, {
            credentials: "include",
            signal: controller.signal
          });
          if (!res.ok) {
            const contentType = res.headers.get("content-type") || "";
            const payload = contentType.includes("application/json") ? await res.json() : await res.text();
            const message = typeof payload === "object" && payload?.error || typeof payload === "string" && payload || `HTTP ${res.status}`;
            throw new Error(humanizeRuntimeStreamError(message, laneState, effectiveTargetMeta));
          }
          setState((prev) => ({ ...prev, status: "live" }));
          const reader = res.body.getReader();
          const decoder = new TextDecoder("utf-8");
          let buffer = "";
          while (!cancelled) {
            const { value, done } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            let idx;
            while ((idx = buffer.indexOf("\n\n")) >= 0) {
              const frame = buffer.slice(0, idx);
              buffer = buffer.slice(idx + 2);
              let eventName = "message";
              const data = [];
              frame.split("\n").forEach((line) => {
                if (line.startsWith("event: ")) eventName = line.slice(7).trim();
                if (line.startsWith("data: ")) data.push(line.slice(6));
              });
              if (!data.length) continue;
              const payload = data.join("\n");
              if (eventName === "runtime-target") {
                try {
                  const parsed = JSON.parse(payload);
                  setState((prev) => ({ ...prev, targetMeta: parsed }));
                } catch {
                }
                continue;
              }
              if (eventName === "runtime-status") {
                try {
                  const parsed = JSON.parse(payload);
                  setState((prev) => ({
                    ...prev,
                    ...(() => {
                      const nextError = humanizeRuntimeStreamError(parsed.error || prev.error, prev.lane, prev.targetMeta || effectiveTargetMeta);
                      return {
                        status: parsed.status === "error" && isRuntimeOfflineError(nextError, prev.lane) ? "offline" : parsed.status || "complete",
                        error: parsed.status === "error" ? nextError : parsed.error ? nextError : prev.error
                      };
                    })()
                  }));
                } catch {
                  setState((prev) => ({ ...prev, status: payload || "complete" }));
                }
                continue;
              }
              setState((prev) => ({
                ...prev,
                lines: [...prev.lines, parseRuntimeLogEntry(payload, prev.targetMeta?.label || effectiveTarget)].slice(-500)
              }));
            }
          }
          setState((prev) => ({
            ...prev,
            status: prev.status === "live" ? "complete" : prev.status
          }));
        } catch (err) {
          if (!cancelled && !controller.signal.aborted) {
            const message = humanizeRuntimeStreamError(err.message, laneState, effectiveTargetMeta);
            setState((prev) => ({
              ...prev,
              status: isRuntimeOfflineError(message, laneState) ? "offline" : "error",
              error: message
            }));
          }
        }
      }
      run();
      return () => {
        cancelled = true;
        controller.abort();
      };
    }, [selectedEnv, selectedTarget, state.defaultTarget, state.lane, state.targets, windowFilter]);
    return state;
  }
  function SplashScreen({ label }) {
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "screen-center", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel splash", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "spinner" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { children: label })
    ] }) });
  }
  function LoginScreen({ onLogin, error }) {
    const [token, setToken] = (0, import_react9.useState)("");
    const [pending, setPending] = (0, import_react9.useState)(false);
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "screen-center", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(
      "form",
      {
        className: "panel login-card",
        onSubmit: async (event) => {
          event.preventDefault();
          setPending(true);
          try {
            await onLogin(token);
          } finally {
            setPending(false);
          }
        },
        children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { type: "text", name: "username", autoComplete: "username", value: "relay-admin", readOnly: true, hidden: true }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(RelayMark, { className: "brand__glyph", title: "Relay mark" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Secure Agent Access" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h1", { className: "login-card__title", children: "Relay Control Room" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { className: "login-card__body", children: "Dashboard sessions are now stored in an HttpOnly cookie. The token never needs to live in browser storage." }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
            "input",
            {
              className: "text-input",
              type: "password",
              autoComplete: "current-password",
              placeholder: "Paste relay token",
              value: token,
              onChange: (event) => setToken(event.target.value)
            }
          ),
          error && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "error-banner", children: error }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "submit", className: "primary-button", disabled: !token || pending, children: pending ? "Authorizing..." : "Enter Dashboard" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "helper-row", children: [
            "Token source",
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "helper-pill", children: "data/token.txt" })
          ] })
        ]
      }
    ) });
  }
  function SourceBadge({ info, compact = false }) {
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: cx2("source-chip", `source-chip--${info.tone}`, compact && "source-chip--compact"), children: info.label });
  }
  function ProjectCommandSelector({ projects, selected, onSelect }) {
    const [open, setOpen] = (0, import_react9.useState)(false);
    const [query, setQuery] = (0, import_react9.useState)("");
    const ref = (0, import_react9.useRef)(null);
    (0, import_react9.useEffect)(() => {
      function onDocumentClick(event) {
        if (ref.current && !ref.current.contains(event.target)) setOpen(false);
      }
      document.addEventListener("mousedown", onDocumentClick);
      return () => document.removeEventListener("mousedown", onDocumentClick);
    }, []);
    const filtered = (0, import_react9.useMemo)(() => {
      const q = query.trim().toLowerCase();
      if (!q) return projects;
      return projects.filter((project) => project.name.toLowerCase().includes(q));
    }, [projects, query]);
    const current = projects.find((project) => project.name === selected);
    const currentRepoInfo = repoProviderInfo(projectRepoURL(current));
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "project-command-selector", ref, children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("button", { type: "button", className: "project-command-selector__button", onClick: () => setOpen((value) => !value), children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "project-command-selector__copy", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "eyebrow", children: "Project Surface" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: current?.name || "Select project" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "project-command-selector__meta", children: current ? `${currentRepoInfo.label} \xB7 ${current.envs.length} environments \xB7 ${current.services.length} services` : "Pick a project, source connection, and deployment surface." })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "project-command-selector__actions", children: [
          current && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SourceBadge, { info: currentRepoInfo, compact: true }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "selector-button__count", children: projects.length })
        ] })
      ] }),
      open && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "selector-menu panel selector-menu--command", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
          "input",
          {
            className: "text-input text-input--compact",
            type: "text",
            placeholder: "Search projects",
            value: query,
            onChange: (event) => setQuery(event.target.value)
          }
        ),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "selector-list", children: [
          filtered.map((project) => {
            const repoInfo = repoProviderInfo(projectRepoURL(project));
            return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(
              "button",
              {
                type: "button",
                className: cx2("selector-item", project.name === selected && "selector-item--active"),
                onClick: () => {
                  onSelect(project.name);
                  setOpen(false);
                },
                children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "selector-item__content", children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
                      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: project.name }),
                      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "selector-item__meta", children: [
                        project.envs.length,
                        " envs \xB7 ",
                        project.services.length,
                        " services"
                      ] })
                    ] }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "selector-item__footer", children: [
                      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SourceBadge, { info: repoInfo, compact: true }),
                      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "selector-item__meta selector-item__host", children: repoInfo.host })
                    ] })
                  ] }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "selector-item__meta", children: project.latestDeployAt ? `${timeAgo(project.latestDeployAt)} ago` : "No deploys" })
                ]
              },
              project.name
            );
          }),
          !filtered.length && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "empty-inline", children: "No projects matched the filter." })
        ] })
      ] })
    ] });
  }
  function MetricCard({ label, value, meta, accent, tone = "neutral" }) {
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: cx2("metric-card", accent && "metric-card--accent", tone !== "neutral" && `metric-card--${tone}`), children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "metric-card__top", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: label }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-card__spark" })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "metric-card__value", children: value }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "metric-card__meta", children: meta })
    ] });
  }
  function ContextBar({ contexts, selected, onSelect }) {
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("section", { className: "context-lanes", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header section-card__header--tight", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Delivery Lanes" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Environment routing" })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "muted", children: "Choose the lane you want to inspect or operate." })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "context-grid", children: contexts.map((context) => {
        const key = deployKey(context.app, context.env, context.branch);
        return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(
          "button",
          {
            type: "button",
            className: cx2("context-card panel", selected === key && "context-card--active"),
            onClick: () => onSelect(key),
            children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "context-card__top", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: cx2("status-chip", deployStatusClass(context.latestDeploy?.status)), children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-dot" }),
                  context.env
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "context-card__branch", children: context.branch })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "context-card__headline", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: context.env }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: liveTargetLabel(context) })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "context-card__url", children: context.previewURL ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: context.previewURL }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "muted", children: "No public URL yet" }) }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills inline-pills--compact", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: trafficModeLabel(context.traffic_mode) }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: engineLabel(context.engine || "docker") }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: rolloutStrategy(context) })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "context-card__footer", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: context.latestDeploy ? deployPhaseText(context.latestDeploy) : "idle" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: context.latestDeploy ? `${timeAgo(context.latestDeploy.created_at)} ago` : "waiting" })
              ] })
            ]
          },
          key
        );
      }) })
    ] });
  }
  function Detail({ label, value, link }) {
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "detail-card", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: label }),
      link ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("a", { className: "link-teal detail-card__value", href: link, target: "_blank", rel: "noreferrer", children: value }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "detail-card__value", children: value })
    ] });
  }
  function ServiceCard({ service }) {
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "row-card row-card--service", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "row-card__title", children: [
          service.name,
          " ",
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "row-card__badge", children: service.type })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "row-card__meta mono", children: service.container }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "row-card__meta mono", children: service.network })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "row-card__actions row-card__actions--column", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "mono connection-pill", children: service.env_key }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: () => navigator.clipboard.writeText(service.env_val), children: "Copy URL" })
      ] })
    ] });
  }
  function OverviewTab({ project, contexts, selectedEnv, services, onOpenDeploy, projectStats, selectedEnvStats }) {
    const selectedPreviewURL = selectedEnv ? computePreviewURL(selectedEnv, selectedEnv.latestDeploy) : "";
    const newTargetURL = targetInspectURL(selectedPreviewURL, "new");
    const oldTargetURL = targetInspectURL(selectedPreviewURL, "old");
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("section", { className: "grid-two", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-card__header", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Environment Surface" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Live contexts" })
        ] }) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "stack-list", children: contexts.map((context) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "row-card", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "row-card__title", children: [
              context.env,
              " / ",
              context.branch
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "row-card__meta", children: context.previewURL ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("a", { className: "link-teal", href: context.previewURL, target: "_blank", rel: "noreferrer", children: context.previewURL }) : "No preview URL" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: rolloutStrategy(context) }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: trafficModeLabel(context.traffic_mode) }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: liveTargetLabel(context) }),
              context.standby_slot && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: oldTargetLabel(context) })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "row-card__actions", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: cx2("status-chip", deployStatusClass(context.latestDeploy?.status)), children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-dot" }),
              context.latestDeploy?.status || "idle"
            ] }),
            !context.stopped && !context.running && context.latestDeploy?.status === "success" && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-chip", children: "container offline" }),
            context.latestDeploy && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: () => onOpenDeploy(context.latestDeploy), children: "Logs" })
          ] })
        ] }, deployKey(context.app, context.env, context.branch))) })
      ] }),
      selectedEnv && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Rollout Flow" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: rolloutStrategy(selectedEnv) })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-chip status-chip--ok", children: liveTargetLabel(selectedEnv) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-chip status-chip--warn", children: oldTargetLabel(selectedEnv) })
          ] })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-grid", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card rollout-card--accent", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "New Target" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: hasSlotRollout(selectedEnv) ? "live" : "single target" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: hasSlotRollout(selectedEnv) ? `All new requests are routed here. ${internalSlotLabel(selectedEnv.active_slot)}` : "This environment is still serving a single target without slot handoff." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Old Target" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: oldTargetLabel(selectedEnv) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: selectedEnv.standby_slot ? `Pinned browsers can still land here until drain completes. ${internalSlotLabel(selectedEnv.standby_slot)}` : hasSlotRollout(selectedEnv) ? "The previous target is already gone. Relay will warm the alternate target on the next deploy." : "No previous target is being kept alive in this environment." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Traffic Policy" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: trafficModeLabel(selectedEnv.traffic_mode) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: selectedEnv.traffic_mode === "session" ? "Relay pins this browser to a target by cookie. Closing the tab or hard-refreshing does not switch it." : "There is no sticky cookie. Once readiness passes, every request moves to the live target." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "New Request Flow" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: "new target only" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: selectedEnv.standby_slot ? selectedEnv.traffic_mode === "session" ? "Only browsers already pinned to the old target stay there during drain. Every new session goes to the new target." : "The old target may still be alive briefly, but new traffic no longer goes there." : hasSlotRollout(selectedEnv) ? "The rollout has already converged on the new target." : "Single-target environments do not split traffic between generations." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Environment Success" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: formatPercent(selectedEnvStats.successRate) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card__meta", children: [
              selectedEnvStats.total,
              " tracked deploys on this lane."
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Average Deploy" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: selectedEnvStats.avgDuration ? prettyDuration(selectedEnvStats.avgDuration) : "n/a" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: "Measured from build start to ready state." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Readiness Gate" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: "60s" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: "Current default candidate warmup timeout." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Old Target Window" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: drainWindowLabel(selectedEnv) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: selectedEnv.standby_slot ? "Relay removes the old target when this drain window expires." : hasSlotRollout(selectedEnv) ? "No old target is currently kept alive." : "Single-target environments do not run a separate drain window." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Traffic Visibility" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: "not tracked" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: "Relay does not yet count how many users or requests remain on the old target versus the new one." })
          ] })
        ] }),
        selectedPreviewURL && hasSlotRollout(selectedEnv) && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-note rollout-note--warn", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: "Inspecting new versus old" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: selectedEnv.traffic_mode === "session" ? "Session mode keeps a sticky cookie, so use these links when you need to pin a browser to the live target or the draining target on purpose." : "Use these links when you need to inspect the live target or the draining target directly from a browser." }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "button-row button-row--compact", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("a", { className: "ghost-button ghost-button--compact", href: newTargetURL, target: "_blank", rel: "noreferrer", children: "Pin browser to new" }),
            selectedEnv.standby_slot && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("a", { className: "ghost-button ghost-button--compact", href: oldTargetURL, target: "_blank", rel: "noreferrer", children: "Pin browser to old" })
          ] })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-note", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: "Weighted canary" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Weighted traffic splitting is not enabled in this build. Relay does full cutover after readiness, then optionally keeps the old target alive only for sticky-session drain." })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-card__header", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Project Telemetry" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Deploy pulse" })
        ] }) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-grid", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Project Success" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: formatPercent(projectStats.successRate) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card__meta", children: [
              projectStats.total,
              " deploys tracked across this project."
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Average Deploy" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: projectStats.avgDuration ? prettyDuration(projectStats.avgDuration) : "n/a" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: "Average rollout time across recorded deploys." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "New/Old Lanes" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: projectStats.blueGreenContexts }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: "Contexts already using the new/old slot handoff flow." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "rollout-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Failure Count" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__value", children: projectStats.failures }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "rollout-card__meta", children: "Failed or errored deploys in the tracked history." })
          ] })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-card__header", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "relay.json Stack" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Companion services" })
        ] }) }),
        services.length ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "stack-list", children: services.map((service) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(ServiceCard, { service }, `${service.env}-${service.branch}-${service.name}`)) }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "code-callout", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { children: "No companion containers are attached to this environment yet." }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("pre", { children: `{
  "project": "${project.name}",
  "services": [
    { "name": "db", "type": "postgres", "version": "16" },
    { "name": "cache", "type": "redis", "version": "7" },
    { "name": "web", "type": "app" }
  ]
}` })
        ] })
      ] }),
      selectedEnv && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card section-card--wide", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Selected Environment" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("h3", { children: [
              selectedEnv.env,
              " / ",
              selectedEnv.branch
            ] })
          ] }),
          selectedEnv.latestDeploy && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: cx2("operation-chip", operationClass(selectedEnv.latestDeploy.source)), children: operationLabel(selectedEnv.latestDeploy.source) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: cx2("status-chip", deployStatusClass(selectedEnv.latestDeploy.status)), children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-dot" }),
              deployPhaseText(selectedEnv.latestDeploy)
            ] })
          ] })
        ] }),
        selectedEnv.latestDeploy?.source === "rollback" && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: cx2("rollout-note", ACTIVE_STATUSES.has(selectedEnv.latestDeploy.status) ? "rollout-note--warn" : selectedEnv.latestDeploy.status === "success" ? "rollout-note--ok" : "rollout-note--danger"), children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: ACTIVE_STATUSES.has(selectedEnv.latestDeploy.status) ? "Rollback running" : selectedEnv.latestDeploy.status === "success" ? "Rollback completed" : "Rollback needs attention" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: ACTIVE_STATUSES.has(selectedEnv.latestDeploy.status) ? `Relay is switching traffic back from ${shortImageTag(selectedEnv.latestDeploy.prev_image_tag || selectedEnv.latestDeploy.prevImage)} to ${shortImageTag(selectedEnv.latestDeploy.image_tag || selectedEnv.latestDeploy.imageTag)}.` : `Latest rollback image ${shortImageTag(selectedEnv.latestDeploy.image_tag || selectedEnv.latestDeploy.imageTag)} with previous live image ${shortImageTag(selectedEnv.latestDeploy.prev_image_tag || selectedEnv.latestDeploy.prevImage)}.` })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "detail-grid", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Engine", value: engineLabel(selectedEnv.engine || "docker") }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Mode", value: selectedEnv.mode || "port" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Rollout Flow", value: rolloutStrategy(selectedEnv) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Traffic Policy", value: trafficModeLabel(selectedEnv.traffic_mode) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "New Target", value: hasSlotRollout(selectedEnv) ? `live (${selectedEnv.active_slot})` : "single target" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Old Target", value: selectedEnv.standby_slot ? `draining (${selectedEnv.standby_slot})` : hasSlotRollout(selectedEnv) ? "removed" : "not staged" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Host Port", value: selectedEnv.host_port || "n/a" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Service Port", value: selectedEnv.service_port || "n/a" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Live Route", value: selectedEnv.previewURL || "No public URL", link: selectedEnv.previewURL }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Saved Route", value: computeConfiguredURL(selectedEnv) || "No saved route", link: computeConfiguredURL(selectedEnv) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Detail, { label: "Repo URL", value: selectedEnv.repo_url || "Not linked" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
            Detail,
            {
              label: "Latest Deploy Duration",
              value: selectedEnv.latestDeploy?.started_at && selectedEnv.latestDeploy?.ended_at ? prettyDuration(new Date(selectedEnv.latestDeploy.ended_at) - new Date(selectedEnv.latestDeploy.started_at)) : "Pending"
            }
          )
        ] })
      ] })
    ] });
  }
  function DeploymentsTab({ project, deploys, envMap, selectedEnv, onOpenDeploy }) {
    const [query, setQuery] = (0, import_react9.useState)("");
    const [statusFilter, setStatusFilter] = (0, import_react9.useState)("all");
    const [laneFilter, setLaneFilter] = (0, import_react9.useState)("all");
    const repoInfo = (0, import_react9.useMemo)(() => repoProviderInfo(projectRepoURL(project)), [project]);
    const statusCounts = (0, import_react9.useMemo)(
      () => ({
        all: deploys.length,
        active: deploys.filter((deploy) => ACTIVE_STATUSES.has(deploy.status)).length,
        success: deploys.filter((deploy) => deploy.status === "success").length,
        issues: deploys.filter((deploy) => deploy.status === "failed" || deploy.status === "error").length
      }),
      [deploys]
    );
    const laneButtons = (0, import_react9.useMemo)(() => {
      const buttons = [{ id: "all", label: "All lanes" }];
      if (selectedEnv) buttons.push({ id: "current", label: `${selectedEnv.env} / ${selectedEnv.branch}` });
      Array.from(new Set(deploys.map((deploy) => deploy.env))).slice(0, 3).forEach((env) => {
        buttons.push({ id: env, label: env });
      });
      return buttons.filter((item, index, items) => items.findIndex((candidate) => candidate.id === item.id) === index);
    }, [deploys, selectedEnv]);
    const filteredDeploys = (0, import_react9.useMemo)(() => {
      const normalizedQuery = query.trim().toLowerCase();
      return deploys.filter((deploy) => {
        const matchesQuery = !normalizedQuery || [
          deploy.id,
          deploy.app,
          deploy.env,
          deploy.branch,
          deploy.commit_sha,
          deploy.image_tag,
          deploy.previous_image_tag
        ].filter(Boolean).some((value) => String(value).toLowerCase().includes(normalizedQuery));
        const matchesStatus = statusFilter === "all" || statusFilter === "active" && ACTIVE_STATUSES.has(deploy.status) || statusFilter === "success" && deploy.status === "success" || statusFilter === "issues" && (deploy.status === "failed" || deploy.status === "error");
        const matchesLane = laneFilter === "all" || laneFilter === "current" && selectedEnv && deploy.env === selectedEnv.env && deploy.branch === selectedEnv.branch || deploy.env === laneFilter;
        return matchesQuery && matchesStatus && matchesLane;
      });
    }, [deploys, laneFilter, query, selectedEnv, statusFilter]);
    const latestDeploy = filteredDeploys[0] || deploys[0] || null;
    const durations = filteredDeploys.map(deployDurationMs).filter(Boolean);
    const avgDuration = durations.length ? Math.round(durations.reduce((sum, value) => sum + value, 0) / durations.length) : 0;
    const successRate = filteredDeploys.length ? filteredDeploys.filter((deploy) => deploy.status === "success").length / Math.max(1, filteredDeploys.length) * 100 : 0;
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("section", { className: "deployments-stage", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card deployments-stage__hero", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Deployment Board" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Release activity" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { className: "muted", children: "Scan recent builds, compare image flow, and jump into a rollout without leaving the project command deck." })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SourceBadge, { info: repoInfo }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: "metric-pill", children: [
            deploys.length,
            " deploys tracked"
          ] }),
          selectedEnv && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: "metric-pill", children: [
            selectedEnv.env,
            " / ",
            selectedEnv.branch
          ] })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployments-toolbar panel", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
          "input",
          {
            className: "text-input deployments-toolbar__search",
            type: "search",
            placeholder: "Search deploy id, branch, commit, or image",
            value: query,
            onChange: (event) => setQuery(event.target.value)
          }
        ),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "segment-list", children: [
          ["all", "All"],
          ["active", "Active"],
          ["success", "Ready"],
          ["issues", "Issues"]
        ].map(([id, label]) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(
          "button",
          {
            type: "button",
            className: cx2("segment-chip", statusFilter === id && "segment-chip--active"),
            onClick: () => setStatusFilter(id),
            children: [
              label,
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: statusCounts[id] })
            ]
          },
          id
        )) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "segment-list", children: laneButtons.map((lane) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
          "button",
          {
            type: "button",
            className: cx2("segment-chip", laneFilter === lane.id && "segment-chip--active"),
            onClick: () => setLaneFilter(lane.id),
            children: lane.label
          },
          lane.id
        )) })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployments-summary-grid", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel ops-stat-card", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Latest Build" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "ops-stat-card__value", children: latestDeploy ? latestDeploy.id.slice(0, 8) : "No deploys" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "ops-stat-card__meta", children: latestDeploy ? `${deployPhaseText(latestDeploy)} \xB7 ${deployDurationLabel(latestDeploy)} \xB7 ${formatDateTime(latestDeploy.created_at)}` : "Trigger a deploy to populate the board." })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel ops-stat-card", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Average Duration" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "ops-stat-card__value", children: avgDuration ? prettyDuration(avgDuration) : "n/a" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "ops-stat-card__meta", children: "Based on the currently filtered deployment set." })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel ops-stat-card", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Success Rate" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "ops-stat-card__value", children: formatPercent(successRate) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "ops-stat-card__meta", children: [
            statusCounts.issues,
            " deploys currently need attention."
          ] })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel ops-stat-card", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Source" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "ops-stat-card__value", children: repoInfo.vendor }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "ops-stat-card__meta", children: repoInfo.host })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-list", children: [
        !filteredDeploys.length && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("section", { className: "panel section-card empty-state empty-state--compact", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "No Matches" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h2", { children: "Nothing landed in this filter set." }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { children: "Adjust the search or status chips to widen the deploy history." })
        ] }),
        filteredDeploys.map((deploy) => {
          const context = envMap.get(deployKey(deploy.app, deploy.env, deploy.branch));
          const preview = computePreviewURL(context, deploy);
          const configured = context ? computeConfiguredURL(context) : "";
          const deployRepo = repoProviderInfo(context?.repo_url || projectRepoURL(project));
          const isCurrentLane = selectedEnv && deploy.env === selectedEnv.env && deploy.branch === selectedEnv.branch;
          return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("article", { className: cx2("panel deployment-entry", isCurrentLane && "deployment-entry--active"), children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-entry__top", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-entry__headline", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "deployment-entry__id", onClick: () => onOpenDeploy(deploy), children: deploy.id.slice(0, 8) }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: cx2("status-chip", deployStatusClass(deploy.status)), children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-dot" }),
                    deployPhaseText(deploy)
                  ] }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: cx2("operation-chip", operationClass(deploy.source)), children: operationLabel(deploy.source) }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SourceBadge, { info: deployRepo, compact: true })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-entry__meta", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { children: [
                    deploy.env,
                    " / ",
                    deploy.branch
                  ] }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: formatDateTime(deploy.created_at) }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "mono", children: formatCommitSHA(deploy.commit_sha) })
                ] })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-entry__actions", children: [
                (preview || configured) && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("a", { className: "ghost-button ghost-button--compact", href: preview || configured, target: "_blank", rel: "noreferrer", children: "Visit" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button ghost-button--compact", onClick: () => onOpenDeploy(deploy), children: "Inspect" })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-entry__grid", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Duration" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: deployDurationLabel(deploy) })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Image" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { className: "mono", children: shortImageTag(deploy.image_tag || deploy.imageTag) })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Previous" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { className: "mono", children: shortImageTag(deploy.previous_image_tag || deploy.prevImage) })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Route" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: context?.public_host || (context?.host_port ? `:${context.host_port}` : "Private") })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-entry__footer", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "deployment-entry__route", children: preview || configured || "Private route" }),
              isCurrentLane && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: "Current lane" })
            ] })
          ] }, deploy.id);
        })
      ] })
    ] });
  }
  function LogsTab({ project, selectedEnv, deploys, envMap, onOpenDeploy }) {
    const [windowFilter, setWindowFilter] = (0, import_react9.useState)("30m");
    const [levelFilter, setLevelFilter] = (0, import_react9.useState)("all");
    const [laneFilter, setLaneFilter] = (0, import_react9.useState)("current");
    const [query, setQuery] = (0, import_react9.useState)("");
    const [selectedTarget, setSelectedTarget] = (0, import_react9.useState)("");
    const scopedDeploys = (0, import_react9.useMemo)(() => {
      return deploys.filter((deploy) => {
        if (laneFilter === "current") {
          return selectedEnv ? deploy.env === selectedEnv.env && deploy.branch === selectedEnv.branch : true;
        }
        if (laneFilter === "all") return true;
        return deploy.env === laneFilter;
      });
    }, [deploys, laneFilter, selectedEnv]);
    const runtimeEnv = (0, import_react9.useMemo)(() => {
      if (laneFilter === "current" || laneFilter === "all") return selectedEnv;
      return Array.from(envMap.values()).find((item) => item.env === laneFilter) || selectedEnv;
    }, [envMap, laneFilter, selectedEnv]);
    const runtime = useRuntimeLogs(runtimeEnv, selectedTarget, windowFilter);
    (0, import_react9.useEffect)(() => {
      if (!runtime.targets.length) {
        setSelectedTarget("");
        return;
      }
      const currentTarget = runtime.targets.find((target) => target.id === selectedTarget);
      const nextTarget = currentTarget?.running ? selectedTarget : runtime.defaultTarget || runtime.targets.find((target) => target.running)?.id || "";
      if (nextTarget !== selectedTarget) {
        setSelectedTarget(nextTarget);
      }
    }, [runtime.defaultTarget, runtime.targets, selectedTarget]);
    const latestBuild = scopedDeploys[0] || deploys[0] || null;
    const latestContext = latestBuild ? envMap.get(deployKey(latestBuild.app, latestBuild.env, latestBuild.branch)) : selectedEnv;
    const latestPreview = latestBuild ? computePreviewURL(latestContext, latestBuild) : "";
    const latestRepo = repoProviderInfo(latestContext?.repo_url || projectRepoURL(project));
    const filteredRuntimeLines = (0, import_react9.useMemo)(
      () => runtime.lines.filter((entry) => runtimeFilterMatches(entry, levelFilter, query)),
      [levelFilter, query, runtime.lines]
    );
    const runtimeCounts = (0, import_react9.useMemo)(
      () => runtime.lines.reduce(
        (acc, entry) => {
          acc.all += 1;
          if (entry.level === "warning") acc.warning += 1;
          if (entry.level === "error") acc.error += 1;
          if (entry.level === "fatal") acc.fatal += 1;
          return acc;
        },
        { all: 0, warning: 0, error: 0, fatal: 0 }
      ),
      [runtime.lines]
    );
    const runningRuntimeTargets = (0, import_react9.useMemo)(() => runtime.targets.filter((target) => target.running), [runtime.targets]);
    const runtimeLaneSummary = (0, import_react9.useMemo)(
      () => describeRuntimeLaneState(runtimeEnv, runtime.lane, runningRuntimeTargets.length),
      [runtime.lane, runtimeEnv, runningRuntimeTargets.length]
    );
    const showRuntimeLaneNotice = Boolean(
      runtimeEnv && runtime.targets.length && (runtime.status === "offline" || runtime.lane.appStopped || !runtime.lane.appRunning || !runtime.loading && runningRuntimeTargets.length === 0)
    );
    const activeRuntimeTarget = runtime.targets.find((target) => target.id === (selectedTarget || runtime.defaultTarget)) || runtime.targetMeta;
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("section", { className: "logs-stage", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-stage__layout", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Card, { className: "logs-filter-rail", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardHeader, { children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Runtime Filters" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(CardTitle, { children: "Log watch" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(CardDescription, { children: "Follow the live app slot, edge proxy, or companion services for the selected lane." })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardContent, { className: "logs-filter-rail__content", children: [
          showRuntimeLaneNotice && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-runtime-state logs-runtime-state--offline", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-runtime-state__top", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-runtime-state__copy", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: runtimeLaneSummary.title }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { children: runtimeLaneSummary.body })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: runtimeLaneSummary.badgeVariant, children: runtimeLaneSummary.badgeLabel })
            ] }),
            latestBuild && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Button, { type: "button", variant: "secondary", className: "logs-runtime-state__action", onClick: () => onOpenDeploy(latestBuild), children: "Open latest build" })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "filter-cluster", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Time Range" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "filter-button-list", children: [
              ["30m", "Last 30 minutes"],
              ["6h", "Last 6 hours"],
              ["24h", "Last 24 hours"]
            ].map(([id, label]) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
              Button,
              {
                variant: windowFilter === id ? "secondary" : "ghost",
                className: cx2("filter-button", windowFilter === id && "filter-button--active"),
                onClick: () => setWindowFilter(id),
                children: label
              },
              id
            )) })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "filter-cluster", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Console Level" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "filter-button-list", children: [
              ["all", "All", runtimeCounts.all],
              ["warning", "Warning", runtimeCounts.warning],
              ["error", "Error", runtimeCounts.error],
              ["fatal", "Fatal", runtimeCounts.fatal]
            ].map(([id, label, count3]) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(
              Button,
              {
                variant: levelFilter === id ? "secondary" : "ghost",
                className: cx2("filter-button filter-button--count", levelFilter === id && "filter-button--active"),
                onClick: () => setLevelFilter(id),
                children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: label }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: count3 })
                ]
              },
              id
            )) })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "filter-cluster", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Lane Scope" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Tabs2, { value: laneFilter, onValueChange: setLaneFilter, className: "logs-filter-tabs", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(TabsList2, { className: "logs-filter-tabs__list", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(TabsTrigger2, { value: "current", children: selectedEnv ? `${selectedEnv.env} / ${selectedEnv.branch}` : "Current lane" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(TabsTrigger2, { value: "prod", children: "Production" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(TabsTrigger2, { value: "preview", children: "Preview" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(TabsTrigger2, { value: "all", children: "All envs" })
            ] }) })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "filter-cluster", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Runtime Targets" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "filter-button-list", children: [
              !runtime.targets.length && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "empty-inline", children: "No runtime targets found for this lane." }),
              runtime.targets.map((target) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(
                Button,
                {
                  variant: selectedTarget === target.id ? "secondary" : "ghost",
                  className: cx2("filter-button filter-button--target", selectedTarget === target.id && "filter-button--active", !target.running && "filter-button--offline"),
                  disabled: !target.running,
                  onClick: () => setSelectedTarget(target.id),
                  children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: target.label }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: target.running ? "success" : "muted", children: target.running ? "live" : "offline" })
                  ]
                },
                target.id
              ))
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-filter-note", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: "Streaming source" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Runtime logs tail the selected engine directly for the live app slot, edge proxy, or selected companion service." })
          ] })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-stage__main", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Card, { className: "logs-console", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardHeader, { className: "logs-console__toolbar", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Input, { value: query, onChange: (event) => setQuery(event.target.value), type: "search", placeholder: "Search runtime logs..." }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: runtime.status === "error" ? "danger" : runtime.status === "live" ? "success" : runtime.status === "offline" ? "warning" : "muted", children: runtime.status === "live" ? "Live stream" : runtime.status === "offline" ? "Runtime offline" : runtime.status }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: "outline", children: windowFilter }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: "outline", children: levelFilter }),
              activeRuntimeTarget?.label && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: "teal", children: activeRuntimeTarget.label })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardContent, { className: "logs-console__content", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-console__header", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Time" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Level" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Host" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Lane" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Messages" })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(ScrollArea2, { className: "logs-console__scroll", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-console__stream", children: [
              runtime.error && runtime.status !== "offline" && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "error-banner", children: runtime.error }),
              !filteredRuntimeLines.length ? /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-console__empty", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "logs-console__glyph" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: runtime.loading ? "Loading runtime targets" : showRuntimeLaneNotice ? runtimeLaneSummary.title : "No runtime logs in this window" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { children: runtime.loading ? "Relay is resolving live containers for the selected lane." : showRuntimeLaneNotice ? runtimeLaneSummary.body : "The selected filter set is quiet right now. Use the build handoff on the right if you need the latest deployment session." })
              ] }) : filteredRuntimeLines.map((entry, index) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-log-row", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "runtime-log-row__time mono", children: entry.time }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: runtimeLogLevelVariant(entry.level), children: entry.level }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "runtime-log-row__host", children: entry.host }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "runtime-log-row__request", children: runtimeEnv ? `${runtimeEnv.env} / ${runtimeEnv.branch}` : "runtime" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-log-row__message mono", children: entry.message })
              ] }, `${index}-${entry.raw.slice(0, 16)}`))
            ] }) })
          ] })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-sidebar", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Card, { className: "logs-sidebar-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardHeader, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Build Log Handoff" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(CardTitle, { children: latestBuild ? latestBuild.id.slice(0, 8) : "No build selected" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(CardDescription, { children: "Keep runtime monitoring on the left, then jump directly into the most recent build log when you need image flow, duration, or rollout detail." })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(CardContent, { children: latestBuild && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_jsx_runtime27.Fragment, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-sidebar-card__grid", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Status" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: deployPhaseText(latestBuild) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Duration" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: deployDurationLabel(latestBuild) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Image" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { className: "mono", children: shortImageTag(latestBuild.image_tag || latestBuild.imageTag) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Route" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: latestContext?.public_host || (latestContext?.host_port ? `:${latestContext.host_port}` : "Private") })
                ] })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SourceBadge, { info: latestRepo, compact: true }),
                latestPreview && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: "outline", children: latestPreview })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Button, { type: "button", variant: "outline", className: "logs-sidebar-card__action", onClick: () => onOpenDeploy(latestBuild), children: "Open build detail" })
            ] }) })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Card, { className: "logs-sidebar-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardHeader, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Recent Build Sessions" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(CardTitle, { children: "Inspect latest rollouts" })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(CardContent, { children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "logs-build-list", children: (scopedDeploys.length ? scopedDeploys : deploys).slice(0, 6).map((deploy) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("button", { type: "button", className: "logs-build-item", onClick: () => onOpenDeploy(deploy), children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-build-item__headline", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: deploy.id.slice(0, 8) }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: cx2("status-chip", deployStatusClass(deploy.status)), children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-dot" }),
                    deploy.status
                  ] })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "logs-build-item__meta", children: [
                  deploy.env,
                  " / ",
                  deploy.branch,
                  " \xB7 ",
                  deployDurationLabel(deploy)
                ] })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: "logs-build-item__time", children: [
                timeAgo(deploy.created_at),
                " ago"
              ] })
            ] }, deploy.id)) }) })
          ] })
        ] })
      ] })
    ] }) });
  }
  function SettingsTab({ project, selectedEnv, services, onUpdated }) {
    const [config, setConfig] = (0, import_react9.useState)(() => buildSettingsConfig(selectedEnv));
    const [secrets, setSecrets] = (0, import_react9.useState)([]);
    const [draftSecret, setDraftSecret] = (0, import_react9.useState)({ key: "", value: "" });
    const [busy, setBusy] = (0, import_react9.useState)(false);
    const [notice, setNotice] = (0, import_react9.useState)(null);
    const [pendingApplyHint, setPendingApplyHint] = (0, import_react9.useState)(false);
    const [companions, setCompanions] = (0, import_react9.useState)([]);
    const [companionBusy, setCompanionBusy] = (0, import_react9.useState)(false);
    const [companionDraft, setCompanionDraft] = (0, import_react9.useState)(() => defaultCompanionDraft("postgres"));
    const [companionEnvText, setCompanionEnvText] = (0, import_react9.useState)("");
    const [companionVolumesText, setCompanionVolumesText] = (0, import_react9.useState)("");
    const [selectedCompanion, setSelectedCompanion] = (0, import_react9.useState)("");
    const [showAdvancedPorts, setShowAdvancedPorts] = (0, import_react9.useState)(() => {
      const servicePort = Number(selectedEnv.service_port) || 0;
      return servicePort > 0 && servicePort !== 3e3;
    });
    const [deleteProjectText, setDeleteProjectText] = (0, import_react9.useState)("");
    const [deleteProjectBusy, setDeleteProjectBusy] = (0, import_react9.useState)(false);
    const [deleteProjectError, setDeleteProjectError] = (0, import_react9.useState)("");
    const webhookURL = `${window.location.origin}/api/webhooks/github`;
    const updateConfig = (0, import_react9.useCallback)((patch) => {
      setConfig((current) => applyEngineConstraints({ ...current, ...patch }));
    }, []);
    const draftConfig = applyEngineConstraints({
      ...selectedEnv,
      ...config,
      host_port: Number(config.host_port) || 0,
      service_port: Number(config.service_port) || 0
    });
    const savedRoute = computeConfiguredURL(selectedEnv);
    const draftRoute = computeConfiguredURL(draftConfig);
    const liveRoute = selectedEnv.latestDeploy?.preview_url || savedRoute;
    const restartRequired = Boolean(savedRoute && liveRoute && savedRoute !== liveRoute);
    const applyRequired = pendingApplyHint || restartRequired;
    const savedEngine = normalizeEngineValue(selectedEnv.engine);
    const draftEngine = normalizeEngineValue(config.engine);
    const draftIsVessel = draftEngine === "vessel";
    const savedHostPort = Number(selectedEnv.host_port) || 0;
    const draftHostPort = Number(config.host_port) || 0;
    const savedServicePort = Number(selectedEnv.service_port) || 0;
    const draftServicePort = Number(config.service_port) || 0;
    const liveMode = selectedEnv.mode || "port";
    const savedMode = selectedEnv.mode || "port";
    const draftMode = config.mode || "port";
    const liveTrafficMode = selectedEnv.traffic_mode || "edge";
    const savedTrafficMode = selectedEnv.traffic_mode || "edge";
    const draftTrafficMode = config.traffic_mode || "edge";
    const configDirty = savedEngine !== draftEngine || (selectedEnv.mode || "port") !== (config.mode || "port") || (selectedEnv.traffic_mode || "edge") !== (config.traffic_mode || "edge") || (selectedEnv.public_host || "") !== (config.public_host || "") || savedHostPort !== draftHostPort || savedServicePort !== draftServicePort;
    const applyState = configDirty ? "draft" : applyRequired ? "restart" : "live";
    const applySummary = applyState === "draft" ? {
      title: "Draft changes are not saved yet",
      detail: "Save and restart to apply the new route or traffic policy to the running app."
    } : applyState === "restart" ? {
      title: "Saved config is waiting for restart",
      detail: "The next deploy already uses the saved settings. Restart only if you want the current container to switch now."
    } : {
      title: "Live app matches saved config",
      detail: "Changes you save later will not affect the running app until restart or the next deploy."
    };
    const primaryActionLabel = configDirty ? "Apply Now" : restartRequired ? "Restart to Apply" : "Restart App";
    const engineOptions = [
      {
        value: "docker",
        title: "Docker",
        summary: "Full Relay feature set: companion services, host routing, and session-sticky rollouts.",
        hint: "Best when you need the most mature runtime behavior."
      },
      {
        value: "vessel",
        title: "Vessel",
        summary: "Experimental native snapshot runner with a faster start path than Docker on the same host.",
        hint: "Session cutovers, companion services, and host-based routing now work here too, but this engine still needs closer verification than Docker."
      }
    ];
    const policyOptions = [
      {
        value: "edge",
        title: "Edge cutover",
        summary: "New requests switch to the new target as soon as deploy health checks pass.",
        hint: "Best for stateless apps and APIs with backward-compatible changes."
      },
      {
        value: "session",
        title: "Session sticky",
        summary: "Browsers stay pinned to one target by cookie until the old target drains.",
        hint: "Safer when old pages must keep talking to old API behavior during rollout."
      }
    ];
    const modeOptions = [
      {
        value: "port",
        title: "Port routing",
        summary: "Expose the app on a direct host port like 2000."
      },
      {
        value: "traefik",
        title: "Host routing",
        summary: "Route by hostname through Traefik instead of a public port."
      }
    ];
    const pendingChanges = [];
    if (savedEngine !== draftEngine) pendingChanges.push(`engine: ${engineLabel(savedEngine)} -> ${engineLabel(draftEngine)}`);
    if (savedMode !== draftMode) pendingChanges.push(`mode: ${savedMode} -> ${draftMode}`);
    if (savedTrafficMode !== draftTrafficMode) pendingChanges.push(`traffic: ${trafficModeLabel(savedTrafficMode)} -> ${trafficModeLabel(draftTrafficMode)}`);
    if ((selectedEnv.public_host || "") !== (config.public_host || "")) pendingChanges.push(`host: ${selectedEnv.public_host || "none"} -> ${config.public_host || "none"}`);
    if (savedHostPort !== draftHostPort) pendingChanges.push(`host port: ${savedHostPort || "none"} -> ${draftHostPort || "none"}`);
    if (savedServicePort !== draftServicePort) pendingChanges.push(`service port: ${savedServicePort || "none"} -> ${draftServicePort || "none"}`);
    const runtimeMatrixRows = [
      { label: "Route", live: liveRoute || "No live route yet", saved: savedRoute || "No saved route", draft: draftRoute || "No draft route" },
      { label: "Engine", live: applyRequired ? "Restart to confirm" : engineLabel(savedEngine), saved: engineLabel(savedEngine), draft: engineLabel(draftEngine) },
      { label: "Mode", live: liveMode, saved: savedMode, draft: draftMode },
      { label: "Traffic", live: trafficModeLabel(liveTrafficMode), saved: trafficModeLabel(savedTrafficMode), draft: trafficModeLabel(draftTrafficMode) }
    ];
    const selectedEngineOption = engineOptions.find((option) => option.value === draftEngine) || engineOptions[0];
    const selectedModeOption = modeOptions.find((option) => option.value === config.mode) || modeOptions[0];
    const selectedPolicyOption = policyOptions.find((option) => option.value === config.traffic_mode) || policyOptions[0];
    const deleteProjectLocked = deleteProjectText.trim() !== project.name;
    (0, import_react9.useEffect)(() => {
      setConfig(buildSettingsConfig(selectedEnv));
      const servicePort = Number(selectedEnv.service_port) || 0;
      setShowAdvancedPorts(servicePort > 0 && servicePort !== 3e3);
      api(`/api/apps/secrets?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}`).then((data) => setSecrets(data || [])).catch(() => setSecrets([]));
      loadCompanions().catch(() => setCompanions([]));
    }, [selectedEnv]);
    (0, import_react9.useEffect)(() => {
      setNotice(null);
      setPendingApplyHint(false);
    }, [selectedEnv.app, selectedEnv.env, selectedEnv.branch]);
    (0, import_react9.useEffect)(() => {
      setDeleteProjectText("");
      setDeleteProjectError("");
    }, [project.name]);
    async function saveConfig() {
      setBusy(true);
      try {
        const saved = await api("/api/apps/config", {
          method: "POST",
          body: JSON.stringify({
            app: selectedEnv.app,
            env: selectedEnv.env,
            branch: selectedEnv.branch,
            ...config,
            host_port: Number(config.host_port) || 0,
            service_port: Number(config.service_port) || 0
          })
        });
        setConfig(buildSettingsConfig(saved));
        setNotice({
          tone: "warn",
          text: "Saved on the agent. The next deploy uses this config automatically. Restart only if you want the current container to switch now."
        });
        setPendingApplyHint(true);
        await onUpdated();
      } finally {
        setBusy(false);
      }
    }
    async function saveAndRestart() {
      setBusy(true);
      try {
        const saved = await api("/api/apps/config", {
          method: "POST",
          body: JSON.stringify({
            app: selectedEnv.app,
            env: selectedEnv.env,
            branch: selectedEnv.branch,
            ...config,
            host_port: Number(config.host_port) || 0,
            service_port: Number(config.service_port) || 0
          })
        });
        setConfig(buildSettingsConfig(saved));
        await api("/api/apps/restart", {
          method: "POST",
          body: JSON.stringify({
            app: selectedEnv.app,
            env: selectedEnv.env,
            branch: selectedEnv.branch
          })
        });
        setNotice({
          tone: "ok",
          text: "Saved and restart sent. Relay is applying the new route and traffic policy to the running app now."
        });
        setPendingApplyHint(false);
        await onUpdated();
      } finally {
        setBusy(false);
      }
    }
    async function control(path) {
      setBusy(true);
      try {
        await api(path, {
          method: "POST",
          body: JSON.stringify({
            app: selectedEnv.app,
            env: selectedEnv.env,
            branch: selectedEnv.branch
          })
        });
        setNotice({
          tone: path.endsWith("/stop") ? "warn" : "ok",
          text: path.endsWith("/restart") ? savedEngine === "vessel" ? "Restart sent. Relay is recycling the saved Vessel snapshot and companion set now." : "Restart sent. The live route should move to the saved port after the refresh finishes." : path.endsWith("/start") ? savedEngine === "vessel" ? "Start sent. Relay is bringing the latest saved Vessel snapshot and enabled companions online now." : "Start sent. Relay is bringing the latest saved image and enabled companions online now." : "Stop sent. The app lane is now kept offline, and future deploys will only build/update until you start it again."
        });
        if (!path.endsWith("/stop")) setPendingApplyHint(false);
        await onUpdated();
      } finally {
        setBusy(false);
      }
    }
    async function addSecret() {
      if (!draftSecret.key || !draftSecret.value) return;
      await api("/api/apps/secrets", {
        method: "POST",
        body: JSON.stringify({
          app: selectedEnv.app,
          env: selectedEnv.env,
          branch: selectedEnv.branch,
          key: draftSecret.key,
          value: draftSecret.value
        })
      });
      const next = await api(`/api/apps/secrets?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}`);
      setSecrets(next || []);
      setDraftSecret({ key: "", value: "" });
    }
    async function removeSecret(key) {
      await api(`/api/apps/secrets?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}&key=${encodeURIComponent(key)}`, {
        method: "DELETE"
      });
      const next = await api(`/api/apps/secrets?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}`);
      setSecrets(next || []);
    }
    async function loadCompanions() {
      const data = await api(`/api/apps/companions?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}`);
      setCompanions(data || []);
      if (data?.length) {
        const current = data.find((item) => item.config?.name === selectedCompanion) || data[0];
        if (current?.config) {
          setSelectedCompanion(current.config.name);
          setCompanionDraft(current.config);
          setCompanionEnvText(envToText(current.config.env));
          setCompanionVolumesText((current.config.volumes || []).join("\n"));
        }
      } else {
        const next = defaultCompanionDraft("postgres");
        setSelectedCompanion("");
        setCompanionDraft(next);
        setCompanionEnvText("");
        setCompanionVolumesText("");
      }
    }
    function hydrateCompanionDraft(config2) {
      const next = {
        ...defaultCompanionDraft(config2.type || "custom"),
        ...config2,
        health: {
          ...defaultCompanionDraft(config2.type || "custom").health,
          ...config2.health || {}
        }
      };
      setSelectedCompanion(next.name || "");
      setCompanionDraft(next);
      setCompanionEnvText(envToText(next.env));
      setCompanionVolumesText((next.volumes || []).join("\n"));
    }
    function startCompanion(kind) {
      const next = defaultCompanionDraft(kind);
      setSelectedCompanion("");
      setCompanionDraft(next);
      setCompanionEnvText(envToText(next.env));
      setCompanionVolumesText((next.volumes || []).join("\n"));
    }
    async function saveCompanion() {
      setCompanionBusy(true);
      try {
        const payload = {
          ...companionDraft,
          env: textToEnv(companionEnvText),
          volumes: companionVolumesText.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
        };
        await api("/api/apps/companions", {
          method: "POST",
          body: JSON.stringify({
            app: selectedEnv.app,
            env: selectedEnv.env,
            branch: selectedEnv.branch,
            config: payload
          })
        });
        await loadCompanions();
        await onUpdated();
      } finally {
        setCompanionBusy(false);
      }
    }
    async function deleteCompanion(name) {
      setCompanionBusy(true);
      try {
        await api(`/api/apps/companions?app=${selectedEnv.app}&env=${selectedEnv.env}&branch=${selectedEnv.branch}&name=${encodeURIComponent(name)}`, {
          method: "DELETE"
        });
        await loadCompanions();
        await onUpdated();
      } finally {
        setCompanionBusy(false);
      }
    }
    async function restartCompanion(name) {
      setCompanionBusy(true);
      try {
        await api("/api/apps/companions/restart", {
          method: "POST",
          body: JSON.stringify({
            app: selectedEnv.app,
            env: selectedEnv.env,
            branch: selectedEnv.branch,
            name
          })
        });
        await loadCompanions();
        await onUpdated();
      } finally {
        setCompanionBusy(false);
      }
    }
    async function deleteProject() {
      if (deleteProjectLocked) return;
      if (!window.confirm(`Delete project "${project.name}" and all deploys, workspaces, companion data, secrets, and runtime state? This cannot be undone.`)) {
        return;
      }
      setDeleteProjectBusy(true);
      setDeleteProjectError("");
      try {
        await api("/api/projects/delete", {
          method: "POST",
          body: JSON.stringify({
            app: project.name
          })
        });
        setDeleteProjectText("");
        await onUpdated();
      } catch (err) {
        setDeleteProjectError(err.message);
      } finally {
        setDeleteProjectBusy(false);
      }
    }
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("section", { className: "grid-two settings-page", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card section-card--wide settings-runtime-card", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-card__header", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header-group", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-icon section-icon--teal", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "18", height: "18", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("rect", { x: "2", y: "3", width: "20", height: "14", rx: "2" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M8 21h8M12 17v4" })
          ] }) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Runtime / Routing" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Server controls" })
          ] })
        ] }) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "settings-runtime-layout", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "settings-runtime-sidebar", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-stage runtime-stage--compact", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-stage__header", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Apply State" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: applySummary.title })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: cx2("runtime-stage__badge", applyState === "live" ? "runtime-stage__badge--ok" : "runtime-stage__badge--warn"), children: applyState === "draft" ? "draft only" : applyState === "restart" ? "restart needed" : "live now" })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { className: "runtime-stage__copy", children: applySummary.detail })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: cx2("runtime-diff-card", pendingChanges.length && "runtime-diff-card--pending"), children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-diff-card__header", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "What Changes Next" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: pendingChanges.length ? `${pendingChanges.length} pending change${pendingChanges.length === 1 ? "" : "s"}` : "No draft changes" })
              ] }),
              pendingChanges.length ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-diff-list", children: pendingChanges.map((item) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "runtime-diff-pill", children: item }, item)) }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { className: "runtime-diff-empty", children: "The form already matches the saved config. Restart only if the running app still has not picked up the saved settings." })
            ] }),
            (notice || applyRequired) && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: cx2("settings-notice", notice?.tone === "ok" && "settings-notice--ok", (notice?.tone === "warn" || applyRequired) && "settings-notice--warn"), children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: applyRequired ? "Restart required" : notice?.tone === "ok" ? "Action sent" : "Config saved" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: applyRequired ? "Saved settings are ahead of the running app. Restart to apply the latest route or traffic policy now." : notice?.text })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "settings-runtime-main", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-matrix-card", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-matrix", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-matrix__head runtime-matrix__head--stub", children: "Config" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-matrix__head", children: "Running" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-matrix__head", children: "Saved" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-matrix__head", children: "Draft" }),
              runtimeMatrixRows.map((row) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_react9.default.Fragment, { children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-matrix__label", children: row.label }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-matrix__cell", children: row.live }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-matrix__cell", children: row.saved }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: cx2("runtime-matrix__cell", row.saved !== row.draft && "runtime-matrix__cell--pending"), children: row.draft })
              ] }, row.label))
            ] }) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-editor-grid", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-control-card", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-control-card__header", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Runtime Engine" }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: selectedEngineOption.title })
                  ] }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "runtime-control-card__meta", children: draftIsVessel ? "experimental lane" : "full feature lane" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-segmented", children: engineOptions.map((option) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
                  "button",
                  {
                    type: "button",
                    className: cx2("runtime-segmented__button", draftEngine === option.value && "runtime-segmented__button--active"),
                    onClick: () => updateConfig({ engine: option.value }),
                    children: option.title
                  },
                  option.value
                )) }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("p", { className: "runtime-control-card__copy", children: [
                  selectedEngineOption.summary,
                  " ",
                  selectedEngineOption.hint
                ] })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-control-card", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-control-card__header", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Routing Mode" }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: selectedModeOption.title })
                  ] }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "runtime-control-card__meta", children: "How users reach this app" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-segmented", children: modeOptions.map((option) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
                  "button",
                  {
                    type: "button",
                    className: cx2("runtime-segmented__button", config.mode === option.value && "runtime-segmented__button--active"),
                    onClick: () => updateConfig({ mode: option.value }),
                    children: option.title
                  },
                  option.value
                )) }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { className: "runtime-control-card__copy", children: selectedModeOption.summary })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-control-card", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "runtime-control-card__header", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Traffic Policy" }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: selectedPolicyOption.title })
                  ] }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "runtime-control-card__meta", children: "What happens during rollout" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "runtime-segmented", children: policyOptions.map((option) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
                  "button",
                  {
                    type: "button",
                    className: cx2("runtime-segmented__button", config.traffic_mode === option.value && "runtime-segmented__button--active"),
                    onClick: () => updateConfig({ traffic_mode: option.value }),
                    children: option.title
                  },
                  option.value
                )) }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("p", { className: "runtime-control-card__copy", children: [
                  selectedPolicyOption.summary,
                  " ",
                  selectedPolicyOption.hint
                ] })
              ] })
            ] }),
            draftIsVessel && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "settings-notice settings-notice--warn", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: "Vessel is experimental" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Relay now routes Vessel lanes through the same lane model as Docker, including session cutover, public host state, and companions. Keep validating custom images and service layouts more closely on this engine." })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "field-grid runtime-form-grid", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Public Host" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: config.public_host, onChange: (e) => updateConfig({ public_host: e.target.value }), placeholder: draftIsVessel ? "Optional but still experimental on Vessel" : "" })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Host Port" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { type: "number", inputMode: "numeric", min: "0", className: "text-input", value: config.host_port, onChange: (e) => updateConfig({ host_port: e.target.value }) })
              ] }),
              showAdvancedPorts && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Service Port" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { type: "number", inputMode: "numeric", min: "0", className: "text-input", value: config.service_port, onChange: (e) => updateConfig({ service_port: e.target.value }) })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "settings-advanced settings-advanced--inline", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button ghost-button--compact", onClick: () => setShowAdvancedPorts((value) => !value), children: showAdvancedPorts ? "Hide Service Port" : "Show Service Port" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "settings-advanced__summary", children: draftIsVessel ? /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_jsx_runtime27.Fragment, { children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "Host Port" }),
                " is where the Vessel edge proxy listens. ",
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "Public Host" }),
                " and session policy now work too, but verify them after switching this lane."
              ] }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_jsx_runtime27.Fragment, { children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "Host Port" }),
                " is public. ",
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "Service Port" }),
                " stays on the app default unless you know it changed."
              ] }) })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "button-row button-row--split", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "primary-button", onClick: configDirty ? saveAndRestart : () => control("/api/apps/restart"), disabled: busy, children: busy ? "Working..." : primaryActionLabel }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: saveConfig, disabled: busy || !configDirty, children: "Save for Later" })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "button-row button-row--compact", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: () => control("/api/apps/stop"), disabled: busy, children: "Stop App" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: () => control("/api/apps/start"), disabled: busy, children: "Start App" })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "settings-footnote", children: draftIsVessel ? /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_jsx_runtime27.Fragment, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: "Vessel note:" }),
              " save the engine switch, then restart or deploy to move this lane onto the latest Vessel snapshot. Routing, session cutover, and companions now follow the same saved lane settings here too, but verify custom workloads after switching."
            ] }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_jsx_runtime27.Fragment, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: "Safe default:" }),
              " change the route or traffic policy, then use ",
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "Apply Now" }),
              ". If you want the next deploy to pick it up without touching the running app, use ",
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "Save for Later" }),
              "."
            ] }) })
          ] })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card settings-meta-card", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-card__header", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header-group", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-icon section-icon--green", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("svg", { width: "18", height: "18", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22" }) }) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "GitHub / Webhooks" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Per-project integration" })
          ] })
        ] }) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Repository URL" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: config.repo_url, onChange: (e) => updateConfig({ repo_url: e.target.value }) })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Webhook URL" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: webhookURL, readOnly: true })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Webhook Secret" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
            "input",
            {
              className: "text-input",
              type: "password",
              autoComplete: "new-password",
              value: config.webhook_secret,
              onChange: (e) => updateConfig({ webhook_secret: e.target.value })
            }
          )
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("p", { className: "muted", children: [
          "When set, the app-specific webhook secret overrides the global ",
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "RELAY_GITHUB_WEBHOOK_SECRET" }),
          "."
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "button-row", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "primary-button", onClick: saveConfig, disabled: busy, children: busy ? "Saving..." : "Save GitHub Settings" }) })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card section-card--wide", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header-group", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-icon section-icon--purple", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "18", height: "18", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("polygon", { points: "12 2 2 7 12 12 22 7 12 2" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("polyline", { points: "2 17 12 22 22 17" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("polyline", { points: "2 12 12 17 22 12" })
            ] }) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Companion Services" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Companion editor" })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: engineLabel(draftEngine) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: "metric-pill", children: [
              companions.length,
              " companions"
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: "agent config overrides repo on deploy" })
          ] })
        ] }),
        draftIsVessel && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "settings-notice settings-notice--warn", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: "Companions on Vessel are newer" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Relay now starts and restarts companions on Vessel lanes too. If a service depends on custom port publishing or volume semantics, verify the running lane after you save." })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "companion-preset-row", children: [
          { kind: "postgres", icon: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "14", height: "14", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("ellipse", { cx: "12", cy: "5", rx: "9", ry: "3" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" })
          ] }) },
          { kind: "redis", icon: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("svg", { width: "14", height: "14", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M13 2L3 14h9l-1 8 10-12h-9l1-8z" }) }) },
          { kind: "worker", icon: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "14", height: "14", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("rect", { x: "4", y: "4", width: "16", height: "16", rx: "2" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("rect", { x: "9", y: "9", width: "6", height: "6" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M9 2v2M15 2v2M9 20v2M15 20v2M20 9h2M20 15h2M2 9h2M2 15h2" })
          ] }) },
          { kind: "custom", icon: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "14", height: "14", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("line", { x1: "12", y1: "5", x2: "12", y2: "19" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("line", { x1: "5", y1: "12", x2: "19", y2: "12" })
          ] }) }
        ].map(({ kind, icon }) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("button", { type: "button", className: "companion-preset-btn", onClick: () => startCompanion(kind), disabled: companionBusy, children: [
          icon,
          "Add ",
          prettyCompanionType(kind)
        ] }, kind)) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-layout", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-list", children: [
            !companions.length && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-empty", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "28", height: "28", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "1.5", strokeLinecap: "round", strokeLinejoin: "round", style: { opacity: 0.3 }, children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("polygon", { points: "12 2 2 7 12 12 22 7 12 2" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("polyline", { points: "2 17 12 22 22 17" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("polyline", { points: "2 12 12 17 22 12" })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "No companions yet" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: draftIsVessel ? "Use a preset above, then verify the live Vessel lane after saving." : "Use a preset above to add your first service." })
            ] }),
            companions.map((item) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(
              "button",
              {
                type: "button",
                className: cx2("companion-list__item", selectedCompanion === item.config.name && "companion-list__item--active"),
                onClick: () => hydrateCompanionDraft(item.config),
                children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-list__title", children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: item.config.name }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: prettyCompanionType(item.config.type) })
                  ] }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-list__meta", children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: item.config.image || `${item.config.type}${item.config.version ? `:${item.config.version}` : ""}` }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: item.source || "agent" })
                  ] }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills inline-pills--compact", children: [
                    item.config.stopped ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-chip", children: "kept off" }) : item.running ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-chip status-chip--ok", children: "running" }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-chip", children: "not running" }),
                    item.config.port ? /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: "metric-pill", children: [
                      "port ",
                      item.config.port
                    ] }) : null,
                    item.config.host_port ? /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: "metric-pill", children: [
                      "host ",
                      item.config.host_port
                    ] }) : null
                  ] })
                ]
              },
              item.config.name
            ))
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-editor", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-form-section", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "companion-form-section__label", children: "Identity" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-editor__grid", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Name" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: companionDraft.name, onChange: (e) => setCompanionDraft({ ...companionDraft, name: e.target.value }) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Type" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("select", { className: "text-input", value: companionDraft.type, onChange: (e) => setCompanionDraft({ ...companionDraft, type: e.target.value }), children: [
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("option", { value: "postgres", children: "postgres" }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("option", { value: "redis", children: "redis" }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("option", { value: "worker", children: "worker" }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("option", { value: "custom", children: "custom" }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("option", { value: "mysql", children: "mysql" }),
                    /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("option", { value: "mongo", children: "mongo" })
                  ] })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Version" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: companionDraft.version || "", onChange: (e) => setCompanionDraft({ ...companionDraft, version: e.target.value }) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field field--two-col", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Image" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: companionDraft.image || "", onChange: (e) => setCompanionDraft({ ...companionDraft, image: e.target.value }) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field field--full", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Command" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: companionDraft.command || "", onChange: (e) => setCompanionDraft({ ...companionDraft, command: e.target.value }) })
                ] })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-form-section", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "companion-form-section__label", children: "Runtime Policy" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "companion-editor__grid", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field field--full", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Desired State" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("select", { className: "text-input", value: companionDraft.stopped ? "stopped" : "running", onChange: (e) => setCompanionDraft({ ...companionDraft, stopped: e.target.value === "stopped" }), children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("option", { value: "running", children: "Running with app" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("option", { value: "stopped", children: "Keep off" })
                ] })
              ] }) })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-form-section", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "companion-form-section__label", children: "Ports" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-editor__grid", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Container Port" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { type: "number", inputMode: "numeric", min: "0", className: "text-input", value: companionDraft.port || 0, onChange: (e) => setCompanionDraft({ ...companionDraft, port: Number(e.target.value) || 0 }) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Host Port" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { type: "number", inputMode: "numeric", min: "0", className: "text-input", value: companionDraft.host_port || 0, onChange: (e) => setCompanionDraft({ ...companionDraft, host_port: Number(e.target.value) || 0 }) })
                ] })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-form-section", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "companion-form-section__label", children: "Config" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-editor__grid", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field field--full", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Environment Variables" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("textarea", { className: "text-input textarea-input", value: companionEnvText, onChange: (e) => setCompanionEnvText(e.target.value), placeholder: "KEY=value\nOTHER=value" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field field--full", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Volumes" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("textarea", { className: "text-input textarea-input", value: companionVolumesText, onChange: (e) => setCompanionVolumesText(e.target.value), placeholder: "/host/path:/container/path\nvolume_name:/data" })
                ] })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-form-section", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "companion-form-section__label", children: "Health Check" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "companion-editor__grid", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field field--full", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Command" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: companionDraft.health?.test || "", onChange: (e) => setCompanionDraft({ ...companionDraft, health: { ...companionDraft.health || {}, test: e.target.value } }), placeholder: "redis-cli ping || exit 1" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Interval (sec)" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { type: "number", inputMode: "numeric", min: "0", className: "text-input", value: companionDraft.health?.interval_seconds || 0, onChange: (e) => setCompanionDraft({ ...companionDraft, health: { ...companionDraft.health || {}, interval_seconds: Number(e.target.value) || 0 } }) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Timeout (sec)" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { type: "number", inputMode: "numeric", min: "0", className: "text-input", value: companionDraft.health?.timeout_seconds || 0, onChange: (e) => setCompanionDraft({ ...companionDraft, health: { ...companionDraft.health || {}, timeout_seconds: Number(e.target.value) || 0 } }) })
                ] })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "button-row button-row--split", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "primary-button", onClick: saveCompanion, disabled: companionBusy, children: companionBusy ? "Saving..." : companionDraft.stopped ? "Save and Keep Off" : "Save Companion" }),
              selectedCompanion ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: () => restartCompanion(selectedCompanion), disabled: companionBusy || companionDraft.stopped, children: "Restart This Companion" }) : null,
              selectedCompanion ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: () => deleteCompanion(selectedCompanion), disabled: companionBusy, children: "Delete Companion" }) : null
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "settings-footnote", children: draftIsVessel ? /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_jsx_runtime27.Fragment, { children: [
              "Companion state persists here for Vessel too. Named volumes become host-backed storage on the agent, and service host ports need extra verification compared with ",
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "Docker" }),
              "."
            ] }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_jsx_runtime27.Fragment, { children: [
              "Companion state now persists here. If a companion is marked ",
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "Keep off" }),
              ", deploys and app restarts leave it off until you turn it back on. Stopping the app also stops its running companions."
            ] }) })
          ] })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-card__header", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header-group", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-icon section-icon--teal", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("svg", { width: "18", height: "18", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" }) }) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Environment Variables" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("h3", { children: [
              "Secrets for ",
              selectedEnv.env,
              " / ",
              selectedEnv.branch
            ] })
          ] })
        ] }) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "field-grid", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Key" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", value: draftSecret.key, onChange: (e) => setDraftSecret({ ...draftSecret, key: e.target.value }) })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Value" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("input", { className: "text-input", type: "password", autoComplete: "new-password", value: draftSecret.value, onChange: (e) => setDraftSecret({ ...draftSecret, value: e.target.value }) })
          ] })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "button-row", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "primary-button", onClick: addSecret, children: "Add Secret" }) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "stack-list", children: [
          !secrets.length && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "empty-inline", children: "No environment variables configured." }),
          secrets.map((secret) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "row-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "row-card__title mono", children: secret.key }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "row-card__meta", children: "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022" })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: () => removeSecret(secret.key), children: "Delete" })
          ] }, secret.key))
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card danger-zone", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-card__header", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header-group", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-icon section-icon--danger", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "18", height: "18", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M3 6h18" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M8 6V4h8v2" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M19 6l-1 14H6L5 6" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M10 11v6M14 11v6" })
          ] }) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Danger Zone" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Delete this project everywhere" })
          ] })
        ] }) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("p", { className: "danger-zone__copy", children: [
          "This deletes every environment for ",
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: project.name }),
          ", plus deploy history, saved config, secrets, companion state, sync sessions, workspaces, logs, edge proxy configs, and stored service volumes."
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("label", { className: "field", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { children: [
            "Type ",
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: project.name }),
            " to confirm"
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
            "input",
            {
              className: "text-input",
              value: deleteProjectText,
              onChange: (e) => setDeleteProjectText(e.target.value),
              placeholder: project.name
            }
          )
        ] }),
        deleteProjectError && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "error-banner", children: deleteProjectError }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "button-row", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
          "button",
          {
            type: "button",
            className: "ghost-button ghost-button--danger",
            onClick: deleteProject,
            disabled: deleteProjectBusy || deleteProjectLocked,
            children: deleteProjectBusy ? "Deleting project..." : "Delete Project and All Data"
          }
        ) })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel section-card", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-card__header", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "section-card__header-group", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "section-icon section-icon--green", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("svg", { width: "18", height: "18", viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("path", { d: "M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("polyline", { points: "3.27 6.96 12 12.01 20.73 6.96" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("line", { x1: "12", y1: "22.08", x2: "12", y2: "12" })
          ] }) }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Companion Containers" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Services on the shared network" })
          ] })
        ] }) }),
        services.length ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "stack-list", children: services.map((service) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(ServiceCard, { service }, `${service.env}-${service.branch}-${service.name}`)) }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "code-callout", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("p", { children: [
            "This environment has no companion services yet. Add a ",
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("code", { children: "relay.json" }),
            " at the repo root."
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("pre", { children: `{
  "project": "${project.name}",
  "services": [
    { "name": "db", "type": "postgres", "version": "16" },
    { "name": "cache", "type": "redis", "version": "7" },
    { "name": "web", "type": "app" }
  ]
}` })
        ] })
      ] })
    ] });
  }
  function LogViewer({ deploy, envInfo, services, onClose }) {
    const [lines, setLines] = (0, import_react9.useState)([]);
    const [status, setStatus] = (0, import_react9.useState)("connecting");
    const logRef = (0, import_react9.useRef)(null);
    const logStats = (0, import_react9.useMemo)(() => buildLogStats(lines), [lines]);
    const preview = computePreviewURL(envInfo, deploy);
    const configuredURL = computeConfiguredURL(envInfo);
    const repoInfo = repoProviderInfo(envInfo?.repo_url);
    (0, import_react9.useEffect)(() => {
      const controller = new AbortController();
      let cancelled = false;
      setLines([]);
      setStatus("connecting");
      async function run() {
        try {
          const res = await fetch(`/api/logs/stream/${deploy.id}`, {
            credentials: "include",
            signal: controller.signal
          });
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          setStatus("live");
          const reader = res.body.getReader();
          const decoder = new TextDecoder("utf-8");
          let buffer = "";
          while (!cancelled) {
            const { value, done } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            let idx;
            while ((idx = buffer.indexOf("\n\n")) >= 0) {
              const frame = buffer.slice(0, idx);
              buffer = buffer.slice(idx + 2);
              let eventName = "message";
              const data = [];
              frame.split("\n").forEach((line) => {
                if (line.startsWith("event: ")) eventName = line.slice(7).trim();
                if (line.startsWith("data: ")) data.push(line.slice(6));
              });
              if (!data.length) continue;
              const payload = data.join("\n");
              if (eventName === "deploy-status") {
                try {
                  const parsed = JSON.parse(payload);
                  setStatus(parsed.status || "complete");
                } catch {
                  setStatus(payload || "complete");
                }
                continue;
              }
              setLines((prev) => [...prev, payload]);
            }
            if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
          }
          setStatus((prev) => prev === "live" ? "complete" : prev);
        } catch (err) {
          if (!cancelled && !controller.signal.aborted) setStatus("error");
        }
      }
      run();
      return () => {
        cancelled = true;
        controller.abort();
      };
    }, [deploy.id]);
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Dialog2, { open: true, onOpenChange: (open) => !open && onClose(), children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(DialogContent2, { className: "modal modal--deploy", "aria-label": `Deployment ${deploy.id}`, children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(DialogHeader, { className: "modal__header", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "eyebrow", children: [
            operationLabel(deploy.source),
            " detail"
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(DialogTitle2, { children: deploy.id }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(DialogDescription2, { className: "inline-pills", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: "outline", className: cx2("operation-chip", operationClass(deploy.source)), children: operationLabel(deploy.source) }),
            (deploy.image_tag || deploy.imageTag) && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Badge, { variant: "outline", className: "mono", children: [
              "to ",
              shortImageTag(deploy.image_tag || deploy.imageTag)
            ] }),
            (deploy.previous_image_tag || deploy.prevImage) && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Badge, { variant: "outline", className: "mono", children: [
              "from ",
              shortImageTag(deploy.previous_image_tag || deploy.prevImage)
            ] })
          ] })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "modal__actions", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: status === "success" || status === "complete" ? "success" : status === "error" ? "danger" : "warning", children: status === "live" ? "streaming" : status }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Button, { type: "button", variant: "outline", onClick: onClose, children: "Close" })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-detail-layout", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-detail-sidebar", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-detail-metrics", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Card, { className: "deployment-metric-card", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardContent, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Created" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "deployment-metric-card__value", children: formatDateTime(deploy.created_at) }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-metric-card__meta", children: [
                timeAgo(deploy.created_at),
                " ago"
              ] })
            ] }) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Card, { className: "deployment-metric-card", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardContent, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Duration" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "deployment-metric-card__value", children: deployDurationLabel(deploy) }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "deployment-metric-card__meta", children: "From build start to completion." })
            ] }) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Card, { className: "deployment-metric-card", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardContent, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Environment" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "deployment-metric-card__value", children: deploy.env }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "deployment-metric-card__meta", children: deploy.branch })
            ] }) }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Card, { className: "deployment-metric-card", children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(CardContent, { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Services" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "deployment-metric-card__value", children: services.length }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "deployment-metric-card__meta", children: "Companion containers in this lane." })
            ] }) })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Accordion2, { type: "single", collapsible: true, defaultValue: "summary", className: "detail-accordion-list", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(AccordionItem2, { value: "summary", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(AccordionTrigger2, { children: "Build Summary" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(AccordionContent2, { children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv-grid", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Commit" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { className: "mono", children: formatCommitSHA(deploy.commit_sha) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Source" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: repoInfo.label })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Image" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { className: "mono", children: shortImageTag(deploy.image_tag || deploy.imageTag) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Previous" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { className: "mono", children: shortImageTag(deploy.previous_image_tag || deploy.prevImage) })
                ] })
              ] }) })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(AccordionItem2, { value: "routing", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(AccordionTrigger2, { children: "Routing Surface" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(AccordionContent2, { children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv-grid", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Preview" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: preview || "Private route" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Saved URL" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: configuredURL || "No configured route" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Engine" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: engineLabel(envInfo?.engine || "docker") })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Mode" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: envInfo?.mode || "port" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Traffic" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: trafficModeLabel(envInfo?.traffic_mode) })
                ] })
              ] }) })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(AccordionItem2, { value: "runtime", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(AccordionTrigger2, { children: "Runtime Notes" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(AccordionContent2, { children: /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv-grid", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Host port" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: envInfo?.host_port || "n/a" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Service port" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: envInfo?.service_port || "n/a" })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Rollout" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: rolloutStrategy(envInfo) })
                ] }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Standby" }),
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: oldTargetLabel(envInfo) })
                ] })
              ] }) })
            ] })
          ] })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "panel deployment-terminal", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-terminal__header", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Build Stream" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h3", { children: "Live output" })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Badge, { variant: "outline", children: [
                logStats.total,
                " lines"
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Badge, { variant: "warning", children: [
                logStats.warnings,
                " warnings"
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(Badge, { variant: "danger", children: [
                logStats.errors,
                " errors"
              ] })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-terminal__summary", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-terminal__status", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(Badge, { variant: status === "success" || status === "complete" ? "success" : status === "error" ? "danger" : "warning", children: status === "live" ? "streaming build output" : status }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SourceBadge, { info: repoInfo, compact: true })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv-grid", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Preview" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: preview || "Private route" })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Image target" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { className: "mono", children: shortImageTag(deploy.image_tag || deploy.imageTag) })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Warnings" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: logStats.warnings })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "deployment-kv", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: "Errors" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("strong", { children: logStats.errors })
              ] })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "log-output mono", ref: logRef, children: [
            !lines.length && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "muted", children: "Connecting to log stream..." }),
            lines.map((line, index) => /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: cx2("log-line", `log-line--${logLineTone(line)}`), children: line }, `${index}-${line.slice(0, 8)}`))
          ] })
        ] })
      ] })
    ] }) });
  }
  function EmptyState() {
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("section", { className: "panel section-card empty-state", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "No Projects Yet" }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h2", { children: "Deploy an app to populate the admin." }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { children: "The project dropdown is now driven by `/api/projects`, so projects appear as soon as Relay has app state." })
    ] });
  }
  function App() {
    const [authState, setAuthState] = (0, import_react9.useState)("checking");
    const [loginError, setLoginError] = (0, import_react9.useState)("");
    const [activeTab, setActiveTab] = (0, import_react9.useState)("overview");
    const [selectedProjectName, setSelectedProjectName] = (0, import_react9.useState)("");
    const [selectedEnvKey, setSelectedEnvKey] = (0, import_react9.useState)("");
    const [selectedDeploy, setSelectedDeploy] = (0, import_react9.useState)(null);
    const [dashboard, refreshDashboard] = useDashboardData(authState === "ready");
    (0, import_react9.useEffect)(() => {
      let cancelled = false;
      api("/api/auth/session").then((session) => {
        if (cancelled) return;
        setAuthState(session?.authenticated ? "ready" : "login");
      }).catch(() => {
        if (!cancelled) setAuthState("login");
      });
      return () => {
        cancelled = true;
      };
    }, []);
    const deploysByContext = (0, import_react9.useMemo)(() => {
      const map = /* @__PURE__ */ new Map();
      for (const deploy of dashboard.deploys) {
        const key = deployKey(deploy.app, deploy.env, deploy.branch);
        if (!map.has(key)) map.set(key, []);
        map.get(key).push(deploy);
      }
      return map;
    }, [dashboard.deploys]);
    const projectOptions = (0, import_react9.useMemo)(() => {
      return dashboard.projects.slice().sort((a, b) => a.name.localeCompare(b.name)).map((project) => ({
        ...project,
        latestDeployAt: Math.max(
          0,
          ...project.envs.map((envInfo) => {
            const deploy = (deploysByContext.get(deployKey(project.name, envInfo.env, envInfo.branch)) || [])[0];
            return deploy ? new Date(deploy.created_at).getTime() : 0;
          })
        )
      })).sort((a, b) => b.latestDeployAt - a.latestDeployAt);
    }, [dashboard.projects, deploysByContext]);
    (0, import_react9.useEffect)(() => {
      if (!projectOptions.length) {
        setSelectedProjectName("");
        return;
      }
      if (!projectOptions.find((item) => item.name === selectedProjectName)) {
        setSelectedProjectName(projectOptions[0].name);
      }
    }, [projectOptions, selectedProjectName]);
    const selectedProject = (0, import_react9.useMemo)(
      () => projectOptions.find((project) => project.name === selectedProjectName) || null,
      [projectOptions, selectedProjectName]
    );
    const envOptions = (0, import_react9.useMemo)(() => {
      if (!selectedProject) return [];
      return selectedProject.envs.slice().sort((a, b) => {
        if (a.env === b.env) return a.branch.localeCompare(b.branch);
        if (a.env === "prod") return -1;
        if (b.env === "prod") return 1;
        return a.env.localeCompare(b.env);
      }).map((envInfo) => {
        const latestDeploy = (deploysByContext.get(deployKey(selectedProject.name, envInfo.env, envInfo.branch)) || [])[0] || null;
        return {
          ...envInfo,
          latestDeploy,
          previewURL: computePreviewURL(envInfo, latestDeploy)
        };
      });
    }, [selectedProject, deploysByContext]);
    (0, import_react9.useEffect)(() => {
      if (!envOptions.length) {
        setSelectedEnvKey("");
        return;
      }
      if (!envOptions.find((item) => deployKey(item.app, item.env, item.branch) === selectedEnvKey)) {
        setSelectedEnvKey(deployKey(envOptions[0].app, envOptions[0].env, envOptions[0].branch));
      }
    }, [envOptions, selectedEnvKey]);
    const selectedEnv = envOptions.find((item) => deployKey(item.app, item.env, item.branch) === selectedEnvKey) || null;
    const projectEnvMap = (0, import_react9.useMemo)(
      () => new Map(envOptions.map((item) => [deployKey(item.app, item.env, item.branch), item])),
      [envOptions]
    );
    const projectServices = (0, import_react9.useMemo)(() => {
      if (!selectedProject || !selectedEnv) return [];
      return (selectedProject.services || []).filter(
        (service) => service.env === selectedEnv.env && service.branch === selectedEnv.branch
      );
    }, [selectedProject, selectedEnv]);
    const projectDeploys = (0, import_react9.useMemo)(() => {
      if (!selectedProject) return [];
      return dashboard.deploys.filter((deploy) => deploy.app === selectedProject.name);
    }, [dashboard.deploys, selectedProject]);
    const selectedEnvDeploys = (0, import_react9.useMemo)(() => {
      if (!selectedEnv) return [];
      return dashboard.deploys.filter(
        (deploy) => deploy.app === selectedEnv.app && deploy.env === selectedEnv.env && deploy.branch === selectedEnv.branch
      );
    }, [dashboard.deploys, selectedEnv]);
    const projectStats = (0, import_react9.useMemo)(() => {
      const completed = projectDeploys.filter((deploy) => deploy.status === "success" || deploy.status === "failed" || deploy.status === "error");
      const successes = completed.filter((deploy) => deploy.status === "success").length;
      const durations = projectDeploys.map(deployDurationMs).filter(Boolean);
      const avgDuration = durations.length ? Math.round(durations.reduce((sum, value) => sum + value, 0) / durations.length) : 0;
      return {
        total: projectDeploys.length,
        failures: completed.filter((deploy) => deploy.status !== "success").length,
        successRate: completed.length ? successes / completed.length * 100 : 0,
        avgDuration,
        blueGreenContexts: (selectedProject?.envs || []).filter((envInfo) => Boolean(envInfo.active_slot)).length
      };
    }, [projectDeploys, selectedProject]);
    const selectedEnvStats = (0, import_react9.useMemo)(() => {
      const completed = selectedEnvDeploys.filter((deploy) => deploy.status === "success" || deploy.status === "failed" || deploy.status === "error");
      const successes = completed.filter((deploy) => deploy.status === "success").length;
      const durations = selectedEnvDeploys.map(deployDurationMs).filter(Boolean);
      return {
        total: selectedEnvDeploys.length,
        successRate: completed.length ? successes / completed.length * 100 : 0,
        avgDuration: durations.length ? Math.round(durations.reduce((sum, value) => sum + value, 0) / durations.length) : 0
      };
    }, [selectedEnvDeploys]);
    const projectRepoInfo = (0, import_react9.useMemo)(() => repoProviderInfo(projectRepoURL(selectedProject)), [selectedProject]);
    const selectedEnvRepoInfo = (0, import_react9.useMemo)(
      () => repoProviderInfo(selectedEnv?.repo_url || projectRepoURL(selectedProject)),
      [selectedEnv, selectedProject]
    );
    const latestProjectDeploy = projectDeploys[0] || null;
    const selectedDeployEnvInfo = selectedDeploy ? projectEnvMap.get(deployKey(selectedDeploy.app, selectedDeploy.env, selectedDeploy.branch)) || null : null;
    const selectedDeployServices = (0, import_react9.useMemo)(() => {
      if (!selectedProject || !selectedDeploy) return [];
      return (selectedProject.services || []).filter(
        (service) => service.env === selectedDeploy.env && service.branch === selectedDeploy.branch
      );
    }, [selectedDeploy, selectedProject]);
    (0, import_react9.useEffect)(() => {
      if (selectedDeploy && selectedProject && selectedDeploy.app !== selectedProject.name) {
        setSelectedDeploy(null);
      }
    }, [selectedDeploy, selectedProject]);
    const selectedPreviewURL = selectedEnv ? computePreviewURL(selectedEnv, selectedEnv.latestDeploy) : "";
    const selectedConfiguredURL = selectedEnv ? computeConfiguredURL(selectedEnv) : "";
    const currentOperationValue = selectedEnv?.latestDeploy ? operationLabel(selectedEnv.latestDeploy.source).toUpperCase() : "IDLE";
    const currentOperationMeta = selectedEnv?.latestDeploy ? deployPhaseText(selectedEnv.latestDeploy) : "waiting for first deploy";
    const currentOperationTone = selectedEnv?.latestDeploy?.status === "failed" || selectedEnv?.latestDeploy?.status === "error" ? "danger" : dashboard.isLive ? "warn" : "teal";
    async function handleLogin(token) {
      setLoginError("");
      try {
        await api("/api/auth/session", {
          method: "POST",
          body: JSON.stringify({ token })
        });
        setAuthState("ready");
      } catch (err) {
        setLoginError(err.message);
      }
    }
    async function handleLogout() {
      await api("/api/auth/session", { method: "DELETE" });
      setAuthState("login");
      setSelectedDeploy(null);
    }
    if (authState === "checking") {
      return /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SplashScreen, { label: "Checking dashboard session" });
    }
    if (authState !== "ready") {
      return /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(LoginScreen, { onLogin: handleLogin, error: loginError });
    }
    if (dashboard.loading) {
      return /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SplashScreen, { label: "Loading projects, services, and deploy history" });
    }
    return /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "app-shell", children: [
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("aside", { className: "sidebar panel", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "sidebar__masthead", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "brand", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(RelayMark, { className: "brand__glyph", title: "Relay mark" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "brand__title", children: "Relay" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Control Room" })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "status-chip status-chip--ok", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-dot" }),
            "Agent live"
          ] })
        ] }),
        selectedProject && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "sidebar-project-card", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Project Surface" }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "sidebar-project-card__title", children: selectedProject.name }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "sidebar-project-card__meta", children: [
            selectedProject.envs.length,
            " environments \xB7 ",
            selectedProject.services.length,
            " services"
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SourceBadge, { info: projectRepoInfo, compact: true }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: "metric-pill", children: [
              formatPercent(projectStats.successRate),
              " success"
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: "metric-pill", children: [
              projectStats.blueGreenContexts,
              " new/old"
            ] })
          ] })
        ] }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "nav-section-label", children: "Navigation" }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("nav", { className: "nav", children: [
          ["overview", "Overview"],
          ["deployments", "Deployments"],
          ["logs", "Logs"],
          ["settings", "Settings"]
        ].map(([id, label]) => /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(
          "button",
          {
            type: "button",
            className: cx2("nav__item", activeTab === id && "nav__item--active"),
            onClick: () => setActiveTab(id),
            children: [
              NAV_ICONS[id],
              label
            ]
          },
          id
        )) }),
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "sidebar__footer", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "sidebar__footer-card", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Operator Hint" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "sidebar__footer-copy", children: "Overview for rollout state, Deployments for history, Logs for live monitoring, Settings for runtime changes." })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: handleLogout, children: "Log out" })
        ] })
      ] }),
      /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("main", { className: "main", children: [
        /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("header", { className: "dashboard-hero panel", children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "dashboard-hero__mast", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "dashboard-hero__identity", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
                ProjectCommandSelector,
                {
                  projects: projectOptions,
                  selected: selectedProjectName,
                  onSelect: (name) => {
                    setSelectedProjectName(name);
                    setActiveTab("overview");
                  }
                }
              ),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "breadcrumb", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { children: selectedProject ? selectedProject.name : "No project" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "breadcrumb__sep", children: "/" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "breadcrumb__active", children: activeTab })
              ] }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("h1", { className: "dashboard-hero__title", children: selectedProject ? selectedProject.name : "Relay Control Room" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("p", { className: "dashboard-hero__copy", children: selectedEnv ? `Operating ${selectedEnv.env} / ${selectedEnv.branch} with ${trafficModeLabel(selectedEnv.traffic_mode)} and ${rolloutStrategy(selectedEnv)}. New requests already land on the live target${selectedEnv.standby_slot ? ", while pinned browsers can still drain on the old target." : "."}` : "Select a project lane to inspect deploy flow, routing, and runtime state." })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "topbar__meta", children: [
              dashboard.isLive ? /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "status-chip status-chip--warn live-badge", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-dot status-dot--pulse" }),
                "Deploying live"
              ] }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button", onClick: refreshDashboard, children: dashboard.refreshing ? "Refreshing..." : "Refresh state" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "metric-pill", children: [
                dashboard.deploys.length,
                " deploys tracked"
              ] })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "dashboard-command-grid", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "command-deck-card", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Source Control" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "command-deck-card__title", children: selectedEnvRepoInfo.label }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "command-deck-card__meta", children: selectedEnvRepoInfo.host }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(SourceBadge, { info: selectedEnvRepoInfo, compact: true }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: selectedProject ? `${selectedProject.envs.length} environments` : "No project selected" })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "command-deck-card", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Deployment Settings" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "command-deck-card__title", children: selectedEnv ? trafficModeLabel(selectedEnv.traffic_mode) : "Waiting for lane" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "command-deck-card__meta", children: selectedEnv ? `${selectedEnv.mode || "port"} mode \xB7 ${selectedConfiguredURL || `host port ${selectedEnv.host_port || "n/a"}`}` : "Select an environment to inspect routing, host, and traffic settings." }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: selectedEnv ? rolloutStrategy(selectedEnv) : "idle" }),
                /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: selectedEnv ? liveTargetLabel(selectedEnv) : "no target" })
              ] })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "command-deck-card", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Latest Build" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "command-deck-card__title", children: latestProjectDeploy ? latestProjectDeploy.id.slice(0, 8) : "No deploy yet" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "command-deck-card__meta", children: latestProjectDeploy ? `${deployPhaseText(latestProjectDeploy)} \xB7 ${deployDurationLabel(latestProjectDeploy)} \xB7 ${formatDateTime(latestProjectDeploy.created_at)}` : "Run a deploy to unlock image flow, duration, and build summary." }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "inline-pills", children: [
                latestProjectDeploy && /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("span", { className: cx2("status-chip", deployStatusClass(latestProjectDeploy.status)), children: [
                  /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "status-dot" }),
                  latestProjectDeploy.status
                ] }),
                selectedPreviewURL && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: selectedPreviewURL })
              ] }),
              latestProjectDeploy && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("button", { type: "button", className: "ghost-button ghost-button--compact command-deck-card__action", onClick: () => setSelectedDeploy(latestProjectDeploy), children: "Inspect build" })
            ] })
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "dashboard-hero__grid", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "dashboard-hero__route", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "eyebrow", children: "Selected Route" }),
              selectedPreviewURL ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("a", { className: "link-teal dashboard-hero__route-link", href: selectedPreviewURL, target: "_blank", rel: "noreferrer", children: selectedPreviewURL }) : /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "dashboard-hero__route-link muted", children: "No public route yet" })
            ] }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "dashboard-hero__stack", children: [
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: selectedEnv ? `${selectedEnv.env} / ${selectedEnv.branch}` : "no lane selected" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: selectedEnv ? trafficModeLabel(selectedEnv.traffic_mode) : "waiting" }),
              /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: selectedEnv ? liveTargetLabel(selectedEnv) : "idle" }),
              selectedEnv?.standby_slot && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("span", { className: "metric-pill", children: oldTargetLabel(selectedEnv) })
            ] })
          ] })
        ] }),
        dashboard.error && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)("div", { className: "error-banner", children: dashboard.error }),
        !selectedProject ? /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(EmptyState, {}) : /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)(import_jsx_runtime27.Fragment, { children: [
          /* @__PURE__ */ (0, import_jsx_runtime27.jsxs)("div", { className: "metric-row", children: [
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(MetricCard, { label: "Environments", value: String(selectedProject.envs.length), meta: "active project lanes", tone: "teal" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(MetricCard, { label: "Services", value: String(selectedProject.services.length), meta: "companion containers", accent: true }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(MetricCard, { label: "Success Rate", value: formatPercent(projectStats.successRate), meta: `${projectStats.total} deploys tracked`, tone: "amber" }),
            /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
              MetricCard,
              {
                label: "Current Operation",
                value: currentOperationValue,
                meta: currentOperationMeta,
                accent: selectedEnv?.latestDeploy?.status === "failed" || selectedEnv?.latestDeploy?.status === "error",
                tone: currentOperationTone
              }
            )
          ] }),
          /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(ContextBar, { contexts: envOptions, selected: selectedEnvKey, onSelect: setSelectedEnvKey }),
          activeTab === "overview" && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
            OverviewTab,
            {
              project: selectedProject,
              contexts: envOptions,
              selectedEnv,
              services: projectServices,
              projectStats,
              selectedEnvStats,
              onOpenDeploy: (deploy) => {
                setSelectedDeploy(deploy);
              }
            }
          ),
          activeTab === "deployments" && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
            DeploymentsTab,
            {
              project: selectedProject,
              deploys: projectDeploys,
              envMap: projectEnvMap,
              selectedEnv,
              onOpenDeploy: setSelectedDeploy
            }
          ),
          activeTab === "logs" && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
            LogsTab,
            {
              project: selectedProject,
              selectedEnv,
              deploys: projectDeploys,
              envMap: projectEnvMap,
              onOpenDeploy: setSelectedDeploy
            }
          ),
          activeTab === "settings" && selectedEnv && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
            SettingsTab,
            {
              project: selectedProject,
              selectedEnv,
              services: projectServices,
              onUpdated: refreshDashboard
            }
          )
        ] }),
        selectedDeploy && /* @__PURE__ */ (0, import_jsx_runtime27.jsx)(
          LogViewer,
          {
            deploy: selectedDeploy,
            envInfo: selectedDeployEnvInfo,
            services: selectedDeployServices,
            onClose: () => setSelectedDeploy(null)
          }
        )
      ] })
    ] });
  }
  (0, import_client.createRoot)(document.getElementById("root")).render(/* @__PURE__ */ (0, import_jsx_runtime27.jsx)(App, {}));
})();
/*! Bundled license information:

react/cjs/react.production.min.js:
  (**
   * @license React
   * react.production.min.js
   *
   * Copyright (c) Facebook, Inc. and its affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)

scheduler/cjs/scheduler.production.min.js:
  (**
   * @license React
   * scheduler.production.min.js
   *
   * Copyright (c) Facebook, Inc. and its affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)

react-dom/cjs/react-dom.production.min.js:
  (**
   * @license React
   * react-dom.production.min.js
   *
   * Copyright (c) Facebook, Inc. and its affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)

react/cjs/react-jsx-runtime.production.min.js:
  (**
   * @license React
   * react-jsx-runtime.production.min.js
   *
   * Copyright (c) Facebook, Inc. and its affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)
*/
