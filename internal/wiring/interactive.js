(() => {
  'use strict';

  const metadata = document.getElementById('cyeam-wire-routes');
  const drawing = metadata.ownerSVGElement;
  const routes = JSON.parse(metadata.textContent);
  const wireLayer = drawing.querySelector('g[id="wires"]');
  const viewport = drawing.viewBox.baseVal;
  const diagramHeight = viewport.height;
  const originalHeight = drawing.height.baseVal.value;
  const uiScale = viewport.width / 900;
  const ns = 'http://www.w3.org/2000/svg';
  const shapes = Array.from(wireLayer.querySelectorAll('[data-cyeam-wire]'));

  function svg(tag, attrs, text) {
    const node = document.createElementNS(ns, tag);
    for (const [key, value] of Object.entries(attrs)) node.setAttribute(key, value);
    if (text !== undefined) node.textContent = text;
    return node;
  }

  const style = svg('style', {}, `
    .cyeam-hit { cursor:pointer; pointer-events:stroke; outline:none }
    .cyeam-hit:focus-visible { stroke:#177b70; stroke-opacity:.24 }
    .cyeam-selected { fill:none; pointer-events:none }
    .cyeam-motion { stroke:#fff; fill:none; pointer-events:none; opacity:0; animation:cyeam-wire-flow .85s ease-out }
    .cyeam-ring { fill:none; pointer-events:none; opacity:0; animation:cyeam-pin-pulse .7s ease-out }
    @keyframes cyeam-wire-flow { from { stroke-dashoffset:90; opacity:.85 } to { stroke-dashoffset:0; opacity:0 } }
    @keyframes cyeam-pin-pulse { from { opacity:1 } to { opacity:0 } }
    @media (prefers-reduced-motion:reduce) { .cyeam-motion,.cyeam-ring { animation:none } }
  `);
  drawing.append(style);
  const selection = svg('g', {'aria-hidden':'true', 'pointer-events':'none'});
  const targets = svg('g', {'aria-label':'接线'});
  wireLayer.after(selection, targets);
  const geometryScale = wireLayer.getCTM().a / drawing.getCTM().a;
  const unit = uiScale / geometryScale;
  const footer = svg('g', {
    transform:`translate(${viewport.x} ${viewport.y + diagramHeight}) scale(${uiScale})`,
    'font-family':'system-ui, sans-serif',
  });
  drawing.append(footer);
  const panel = svg('rect', {x:0, y:0, width:900, fill:'#f3f6f8'});
  const message = svg('text', {x:20, y:26, fill:'#23384a', 'font-size':16, 'font-weight':650});
  const labels = svg('g', {fill:'#314958', 'font-size':14, role:'status', 'aria-live':'polite'});
  footer.append(panel, message, labels);

  function pinName(pin) {
    return pin.pin ? pin.part + ' · ' + pin.pin : pin.part;
  }

  function wireCopy(index, className) {
    const copy = shapes[index].cloneNode(false);
    copy.removeAttribute('data-cyeam-wire');
    copy.setAttribute('class', className);
    return copy;
  }

  function updateFooter(route, color) {
    const descriptions = route ? route.pins.map(pinName) : [];
    labels.replaceChildren();
    // One row per terminal keeps the device names and pin labels readable when a net branches.
    descriptions.forEach((name, index) => {
      const row = svg('text', {x:35, y:54 + index*22}, name);
      if (name.length > 105) {
        row.setAttribute('textLength', '830');
        row.setAttribute('lengthAdjust', 'spacingAndGlyphs');
      }
      labels.append(svg('circle', {cx:22, cy:49+index*22, r:3.5, fill:color}), row);
    });
    message.textContent = route ? `接线端点  ·  ${descriptions.length} 个` : '点一根线查看接线端点';
    const height = descriptions.length ? 50+descriptions.length*22 : 45;
    panel.setAttribute('height', height);
    viewport.height = diagramHeight + height*uiScale;
    drawing.setAttribute('height', originalHeight*viewport.height/diagramHeight);
  }

  function select(route, clicked) {
    selection.replaceChildren();
    wireLayer.style.opacity = route ? '.28' : '';
    drawing.toggleAttribute('data-cyeam-selected', Boolean(route));
    if (!route) {
      updateFooter();
      return;
    }
    const color = shapes[clicked].getAttribute('stroke');
    for (const index of route.wires) {
      const copy = wireCopy(index, 'cyeam-selected');
      copy.setAttribute('stroke-width', unit*(index === clicked ? 4.6 : 3.4));
      selection.append(copy);
    }
    for (const index of route.wires) {
      const copy = wireCopy(index, 'cyeam-motion');
      copy.setAttribute('stroke-width', unit*1.1);
      copy.setAttribute('stroke-dasharray', `${unit*4} ${unit*7}`);
      selection.append(copy);
    }
    for (const pin of route.pins) {
      const [cx, cy] = pin.at;
      selection.append(
        svg('circle', {cx,cy,r:unit*5.5,fill:'#fff',stroke:color,'stroke-width':unit*1.8}),
        svg('circle', {cx,cy,r:unit*2.1,fill:color}),
        svg('circle', {cx,cy,r:unit*8,stroke:color,'stroke-width':unit*2,class:'cyeam-ring'}),
      );
    }
    updateFooter(route, color);
  }

  for (const route of routes) {
    const label = route.pins.map(pinName).join(' ↔ ') || '接线';
    for (const [order, index] of route.wires.entries()) {
      const hit = wireCopy(index, 'cyeam-hit');
      hit.setAttribute('stroke', 'transparent');
      hit.setAttribute('stroke-width', unit*11);
      hit.dataset.cyeamHit = index;
      hit.append(svg('title', {}, label));
      hit.addEventListener('click', event => {
        event.stopPropagation();
        select(route, index);
      });
      if (order === 0) {
        hit.setAttribute('tabindex', '0');
        hit.setAttribute('role', 'button');
        hit.setAttribute('aria-label', label);
        hit.addEventListener('keydown', event => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            select(route, index);
          }
        });
      }
      targets.append(hit);
    }
  }
  drawing.addEventListener('click', () => select());
  drawing.addEventListener('keydown', event => {
    if (event.key === 'Escape') select();
  });
  updateFooter();
})();
