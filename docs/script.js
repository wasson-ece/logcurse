// logcurse help site — minimal JS for nav + typing effect

(function () {
  "use strict";

  // Sticky nav shadow on scroll
  const nav = document.getElementById("nav");
  window.addEventListener("scroll", function () {
    nav.classList.toggle("scrolled", window.scrollY > 20);
  }, { passive: true });

  // Active nav link tracking
  const sections = document.querySelectorAll(".section[id]");
  const navLinks = document.querySelectorAll(".nav-links a");

  const observer = new IntersectionObserver(
    function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          const id = entry.target.id;
          navLinks.forEach(function (link) {
            link.classList.toggle("active", link.getAttribute("href") === "#" + id);
          });
        }
      });
    },
    { rootMargin: "-30% 0px -60% 0px" }
  );

  sections.forEach(function (section) {
    observer.observe(section);
  });

  // Typing effect in hero
  var commands = [
    "logcurse -n '140,160p' server.log",
    "logcurse server.log",
    "logcurse --serve server.log",
  ];
  var typed = document.getElementById("typed");
  var cmdIndex = 0;
  var charIndex = 0;
  var deleting = false;
  var pauseTimer = null;

  function tick() {
    var cmd = commands[cmdIndex];

    if (!deleting) {
      typed.textContent = cmd.slice(0, charIndex + 1);
      charIndex++;
      if (charIndex >= cmd.length) {
        pauseTimer = setTimeout(function () {
          deleting = true;
          tick();
        }, 2400);
        return;
      }
      setTimeout(tick, 55 + Math.random() * 40);
    } else {
      typed.textContent = cmd.slice(0, charIndex);
      charIndex--;
      if (charIndex < 0) {
        deleting = false;
        charIndex = 0;
        cmdIndex = (cmdIndex + 1) % commands.length;
        setTimeout(tick, 500);
        return;
      }
      setTimeout(tick, 25);
    }
  }

  setTimeout(tick, 800);
})();
