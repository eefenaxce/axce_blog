// Halo-compatible widget loader.
// The actual comment and search UI pages are provided by the backend under
// /apis/api.plugin.halo.run/v1alpha1/plugins/PluginCommentWidget and
// /apis/api.plugin.halo.run/v1alpha1/plugins/PluginSearchWidget.
(function () {
  'use strict';

  // Halo widget pages are served at absolute /apis/... paths; they do not use
  // the theme base path which is reserved for theme static assets.
  function apiUrl(path) {
    if (!path.startsWith('/')) path = '/' + path;
    return path;
  }

  // Auto-resize comment iframes based on messages from the embedded page.
  window.addEventListener('message', function (e) {
    if (!e.data || typeof e.data !== 'object') return;
    if (e.data.type === 'halo-comment-resize') {
      const iframe = document.getElementById('halo-comment-iframe-' + e.data.frameId);
      if (iframe && typeof e.data.height === 'number') {
        iframe.style.height = e.data.height + 'px';
      }
    }
  });

  /* ============================================================
     halo-comment custom element
     ============================================================ */
  class HaloComment extends HTMLElement {
    connectedCallback() {
      const group = this.getAttribute('group') || 'content.halo.run';
      const kind = this.getAttribute('kind') || 'Post';
      const name = this.getAttribute('name') || '';
      const frameId = 'hc-' + Math.random().toString(36).slice(2);

      this.innerHTML = '';
      if (!name) {
        this.innerHTML = '<div style="padding:24px 0;text-align:center;color:#999">博主已关闭当前页面的评论</div>';
        return;
      }

      const iframe = document.createElement('iframe');
      iframe.id = 'halo-comment-iframe-' + frameId;
      let src = apiUrl('/apis/api.plugin.halo.run/v1alpha1/plugins/PluginCommentWidget/comments/render')
        + '?group=' + encodeURIComponent(group)
        + '&kind=' + encodeURIComponent(kind)
        + '&name=' + encodeURIComponent(name)
        + '&frameId=' + encodeURIComponent(frameId);
      const token = localStorage.getItem('auth_token');
      if (token) {
        src += '&token=' + encodeURIComponent(token);
      }
      iframe.src = src;
      iframe.style.width = '100%';
      iframe.style.border = 'none';
      iframe.style.minHeight = '160px';
      iframe.setAttribute('scrolling', 'no');
      iframe.setAttribute('title', '评论区');
      this.appendChild(iframe);
    }
  }

  if (!customElements.get('halo-comment')) {
    customElements.define('halo-comment', HaloComment);
  }

  /* ============================================================
     SearchWidget popup
     ============================================================ */
  window.__haloCloseSearchWidget = function () {
    const modal = document.getElementById('halo-search-widget-modal');
    if (modal) modal.remove();
  };

  if (typeof window.SearchWidget === 'undefined') {
    window.SearchWidget = {
      open: function () {
        if (document.getElementById('halo-search-widget-modal')) return;

        const modal = document.createElement('div');
        modal.id = 'halo-search-widget-modal';
        modal.innerHTML = `
          <style>
            #halo-search-widget-modal {
              position: fixed;
              inset: 0;
              z-index: 9999;
              background: rgba(0,0,0,0.45);
              display: flex;
              align-items: flex-start;
              justify-content: center;
              padding-top: 10vh;
            }
            #halo-search-widget-modal iframe {
              width: min(640px, 92vw);
              height: min(520px, 80vh);
              border: none;
              border-radius: 12px;
              box-shadow: 0 20px 60px rgba(0,0,0,0.25);
              background: var(--halo-search-widget-base-bg-color, #fff);
            }
          </style>
        `;
        const iframe = document.createElement('iframe');
        iframe.src = apiUrl('/apis/api.plugin.halo.run/v1alpha1/plugins/PluginSearchWidget/render');
        iframe.setAttribute('title', '搜索');
        modal.appendChild(iframe);
        document.body.appendChild(modal);

        modal.addEventListener('click', function (e) {
          if (e.target === modal) window.__haloCloseSearchWidget();
        });
      },
    };
  }
})();
