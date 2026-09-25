// Counts this view, then rolls the counter to the live number: the page itself
// may have been cached minutes ago.
function countView() {
  const post = document.querySelector("[data-slug]");
  const counter = document.querySelector(".views-count");
  if (!post || !counter) return;

  fetch(`/blogs/${encodeURIComponent(post.dataset.slug)}/view`, { method: "POST" })
    .then((response) => (response.ok ? response.json() : null))
    .then((body) => {
      if (body && body.views > 0) roll(counter, body.views);
    })
    .catch(() => {});
}

function roll(counter, views) {
  const reels = [...String(views)].map((digit) => {
    const steps = Number(digit);
    const strip = Array.from({ length: steps + 1 }, (_, n) => `<span>${n}</span>`).join("");
    return `<span class="slot-reel spin" style="--steps:${steps}"><span class="slot-strip">${strip}</span></span>`;
  });
  counter.innerHTML = reels.join("");
  counter.setAttribute("aria-label", String(views));
}

// Marks the heading being read in the table of contents.
function followHeadings() {
  const links = new Map(
    [...document.querySelectorAll(".toc-link")].map((link) => [decodeURIComponent(link.hash.slice(1)), link]),
  );
  if (!links.size) return;

  let active = null;
  const observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (!entry.isIntersecting) continue;
        const link = links.get(entry.target.id);
        if (!link) continue;
        active?.classList.remove("active");
        link.classList.add("active");
        active = link;
      }
    },
    { rootMargin: "0px 0px -75% 0px", threshold: 0 },
  );
  for (const id of links.keys()) {
    const heading = document.getElementById(id);
    if (heading) observer.observe(heading);
  }
}

countView();
followHeadings();
