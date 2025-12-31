// Advanced debug script - Run this in Chrome DevTools Console
console.log('=== Complete Style Debug ===');

// Check all elements
const elements = [
    { name: 'HTML', el: document.documentElement },
    { name: 'Body', el: document.body },
    { name: 'Container', el: document.querySelector('.container') },
    { name: 'Content Area', el: document.querySelector('.content-area') },
    { name: 'Tab Area', el: document.querySelector('.tab-area') }
];

elements.forEach(({ name, el }) => {
    if (el) {
        const styles = getComputedStyle(el);
        console.log(`\n${name}:`);
        console.log('  Background:', styles.backgroundColor);
        console.log('  Border:', styles.border);
        console.log('  Outline:', styles.outline);
        console.log('  Box-shadow:', styles.boxShadow);
        console.log('  Classes:', el.className);
    }
});

// Check for any white/light colored elements
console.log('\n=== Looking for white elements ===');
const allElements = document.querySelectorAll('*');
let whiteElements = [];
allElements.forEach(el => {
    const bg = getComputedStyle(el).backgroundColor;
    if (bg.includes('255, 255, 255') || bg.includes('rgb(255')) {
        whiteElements.push({
            tag: el.tagName,
            class: el.className,
            bg: bg
        });
    }
});
console.log('White elements found:', whiteElements);

// Try to force everything black
console.log('\n=== Forcing all backgrounds to black ===');
document.documentElement.style.background = '#000';
document.body.style.background = '#000';
if (document.querySelector('.container')) {
    document.querySelector('.container').style.background = '#000';
}
