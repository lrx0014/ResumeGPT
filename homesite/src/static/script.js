// ResumeGPT marketing site — vanilla JS, no dependencies.
(function () {
  "use strict";

  var yearEl = document.getElementById("year");
  if (yearEl) yearEl.textContent = new Date().getFullYear();

  var navToggle = document.getElementById("navToggle");
  var navLinks = document.getElementById("navLinks");
  if (navToggle && navLinks) {
    navToggle.addEventListener("click", function () {
      var isOpen = navLinks.classList.toggle("open");
      navToggle.setAttribute("aria-expanded", String(isOpen));
    });
    navLinks.querySelectorAll("a").forEach(function (link) {
      link.addEventListener("click", function () {
        navLinks.classList.remove("open");
        navToggle.setAttribute("aria-expanded", "false");
      });
    });
  }

  var langToggle = document.getElementById("langToggle");
  var langMenu = document.getElementById("langMenu");
  if (langToggle && langMenu) {
    langToggle.addEventListener("click", function (event) {
      event.stopPropagation();
      var isOpen = langMenu.classList.toggle("open");
      langToggle.setAttribute("aria-expanded", String(isOpen));
    });
    document.addEventListener("click", function (event) {
      if (!langMenu.contains(event.target) && event.target !== langToggle) {
        langMenu.classList.remove("open");
        langToggle.setAttribute("aria-expanded", "false");
      }
    });
    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape") {
        langMenu.classList.remove("open");
        langToggle.setAttribute("aria-expanded", "false");
      }
    });
  }

  var lightbox = document.getElementById("lightbox");
  var lightboxImg = document.getElementById("lightboxImg");
  var lightboxClose = document.getElementById("lightboxClose");
  var shotGrid = document.getElementById("shotGrid");

  function openLightbox(src, alt) {
    if (!lightbox || !lightboxImg) return;
    lightboxImg.src = src;
    lightboxImg.alt = alt || "";
    lightbox.classList.add("open");
    document.body.style.overflow = "hidden";
  }

  function closeLightbox() {
    if (!lightbox) return;
    lightbox.classList.remove("open");
    document.body.style.overflow = "";
  }

  if (shotGrid) {
    shotGrid.querySelectorAll(".shot").forEach(function (button) {
      button.addEventListener("click", function () {
        var full = button.getAttribute("data-full");
        var caption = button.getAttribute("data-caption");
        openLightbox(full, caption);
      });
    });
  }

  if (lightboxClose) lightboxClose.addEventListener("click", closeLightbox);
  if (lightbox) {
    lightbox.addEventListener("click", function (event) {
      if (event.target === lightbox) closeLightbox();
    });
  }
  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape") closeLightbox();
  });
})();
