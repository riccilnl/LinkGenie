// Debug script - paste this in Chrome DevTools Console to check theme status
console.log('=== Theme Debug Info ===');
console.log('HTML element classes:', document.documentElement.className);
console.log('Body element classes:', document.body.className);
console.log('HTML background color:', getComputedStyle(document.documentElement).backgroundColor);
console.log('Body background color:', getComputedStyle(document.body).backgroundColor);

// Check storage
chrome.storage.local.get(['theme'], (result) => {
    console.log('Stored theme:', result.theme);
});

// Force apply dark theme
console.log('\n=== Forcing Dark Theme ===');
document.documentElement.classList.add('dark-theme');
document.body.classList.add('dark-theme');
console.log('Dark theme applied. Check if white border is gone.');
