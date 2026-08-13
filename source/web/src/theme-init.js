(() => {
  const key = "bonghos.theme";
  const choice = localStorage.getItem(key) || "system";
  const dark = choice === "dark" || (choice === "system" && matchMedia("(prefers-color-scheme: dark)").matches);
  document.documentElement.dataset.theme = choice;
  document.documentElement.dataset.resolvedTheme = dark ? "dark" : "light";
})();
