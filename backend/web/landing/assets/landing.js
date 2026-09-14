// Landing: nav, filter tema, dialog demo, scroll reveal. Vanilla, tanpa dependensi.
(function () {
  'use strict'
  var doc = document
  var reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches

  // --- nav: garis bawah saat scroll + tutup menu mobile setelah klik ---
  var nav = doc.querySelector('.nav')
  if (nav) {
    var onScroll = function () { nav.classList.toggle('is-scrolled', window.scrollY > 8) }
    window.addEventListener('scroll', onScroll, { passive: true })
    onScroll()
  }
  var menu = doc.querySelector('.menu')
  if (menu) {
    menu.addEventListener('click', function (e) {
      if (e.target.closest('a')) menu.removeAttribute('open')
    })
    doc.addEventListener('click', function (e) {
      if (menu.open && !menu.contains(e.target)) menu.removeAttribute('open')
    })
    doc.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && menu.open) {
        menu.removeAttribute('open')
        menu.querySelector('summary').focus()
      }
    })
  }

  // --- filter kategori tema ---
  var list = doc.querySelector('[data-themes]')
  var filterNav = doc.querySelector('[data-filter-nav]')
  if (list && filterNav) {
    var cards = Array.prototype.slice.call(list.querySelectorAll('.theme'))
    var limit = parseInt(list.getAttribute('data-limit') || '0', 10) // >0 = halaman utama
    var empty = doc.querySelector('[data-empty]')
    var count = doc.querySelector('[data-count]')
    var more = doc.querySelector('[data-more-link]')
    var moreText = more ? more.innerHTML : ''

    var apply = function (cat, label, push) {
      var shown = 0
      cards.forEach(function (card) {
        var match = !cat || card.getAttribute('data-cat') === cat
        // Halaman utama: tanpa filter tampil `limit` kartu pertama; dengan filter tampil semua yang cocok.
        var visible = match && (!limit || cat || shown < limit)
        card.hidden = !visible
        if (visible) shown++
      })
      list.classList.toggle('is-filtered', true)
      Array.prototype.forEach.call(filterNav.querySelectorAll('[data-filter]'), function (a) {
        if (a.getAttribute('data-filter') === cat) a.setAttribute('aria-current', 'true')
        else a.removeAttribute('aria-current')
      })
      if (empty) empty.hidden = shown > 0
      if (count) count.textContent = 'Menampilkan ' + shown + ' tema' + (cat ? ' kategori ' + label : '')
      if (more) {
        more.href = cat ? '/tema?kategori=' + encodeURIComponent(cat) : '/tema'
        if (cat) more.firstChild.textContent = 'Lihat tema ' + label + ' '
        else more.innerHTML = moreText
      }
      if (push && !limit && window.history && history.replaceState) {
        history.replaceState(null, '', cat ? '/tema?kategori=' + encodeURIComponent(cat) : '/tema')
      }
    }

    filterNav.addEventListener('click', function (e) {
      var a = e.target.closest('[data-filter]')
      if (!a || e.metaKey || e.ctrlKey || e.shiftKey || e.button > 0) return
      e.preventDefault()
      var label = (a.childNodes[0] && a.childNodes[0].textContent || '').trim()
      apply(a.getAttribute('data-filter'), label, true)
    })
  }

  // --- dialog demo tema ---
  var dialog = doc.querySelector('[data-demo-dialog]')
  if (dialog && typeof dialog.showModal === 'function') {
    var frame = dialog.querySelector('[data-demo-frame]')
    var title = dialog.querySelector('[data-demo-title]')
    var openTab = dialog.querySelector('[data-demo-open]')
    var use = dialog.querySelector('[data-demo-use]')
    var opener = null

    doc.addEventListener('click', function (e) {
      var link = e.target.closest('[data-demo]')
      if (!link || e.metaKey || e.ctrlKey || e.shiftKey || e.button > 0) return
      e.preventDefault()
      opener = link
      var href = link.getAttribute('href')
      var name = link.getAttribute('data-name') || 'Tema'
      title.textContent = name
      frame.title = 'Pratinjau tema ' + name
      frame.src = href
      openTab.href = href
      if (link.getAttribute('data-use')) use.href = link.getAttribute('data-use')
      dialog.showModal()
      doc.documentElement.style.overflow = 'hidden'
    })
    var close = function () { dialog.close() }
    dialog.querySelector('[data-demo-close]').addEventListener('click', close)
    dialog.addEventListener('click', function (e) { if (e.target === dialog) close() })
    dialog.addEventListener('close', function () {
      frame.src = 'about:blank' // hentikan musik/animasi tema
      doc.documentElement.style.overflow = ''
      if (opener) opener.focus()
    })
  }

  // --- scroll reveal (elemen di luar layar saja, supaya tidak ada kedipan) ---
  var items = doc.querySelectorAll('[data-reveal]')
  if (!reduce && 'IntersectionObserver' in window && items.length) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) {
        var el = en.target
        if (!el.__seen) {
          el.__seen = true
          if (en.isIntersecting) { io.unobserve(el); return }
          el.classList.add('reveal-pending')
          return
        }
        if (en.isIntersecting) {
          el.classList.add('reveal-in')
          el.classList.remove('reveal-pending')
          io.unobserve(el)
        }
      })
    }, { rootMargin: '0px 0px -8% 0px', threshold: 0.08 })
    Array.prototype.forEach.call(items, function (el) {
      // stagger kecil untuk item bersaudara
      var sib = el.parentElement ? Array.prototype.indexOf.call(el.parentElement.children, el) : 0
      if (sib > 0) el.style.setProperty('--delay', Math.min(sib, 5) * 60 + 'ms')
      io.observe(el)
    })
  }
})()
