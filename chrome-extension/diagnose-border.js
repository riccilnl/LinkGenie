// 完整的白边框诊断脚本
// 在Chrome侧边栏的开发者工具Console中运行

console.log('=== 白边框诊断 ===\n');

// 1. 检查所有关键元素的样式
const elements = [
    { name: 'HTML', el: document.documentElement },
    { name: 'Body', el: document.body },
    { name: 'Container', el: document.querySelector('.container') }
];

elements.forEach(({ name, el }) => {
    if (el) {
        const styles = getComputedStyle(el);
        console.log(`${name}:`);
        console.log('  Background:', styles.backgroundColor);
        console.log('  Margin:', styles.margin);
        console.log('  Padding:', styles.padding);
        console.log('  Border:', styles.border);
        console.log('  Width:', styles.width);
        console.log('  Height:', styles.height);
        console.log('  Color-scheme:', styles.colorScheme);
        console.log('');
    }
});

// 2. 检查是否有白色元素
console.log('=== 查找白色元素 ===');
let whiteCount = 0;
document.querySelectorAll('*').forEach(el => {
    const bg = getComputedStyle(el).backgroundColor;
    if (bg.includes('255, 255, 255') || bg === 'rgb(255, 255, 255)') {
        whiteCount++;
        if (whiteCount <= 5) { // 只显示前5个
            console.log(`- ${el.tagName}.${el.className}: ${bg}`);
        }
    }
});
console.log(`共找到 ${whiteCount} 个白色元素\n`);

// 3. 检查主题类
console.log('=== 主题状态 ===');
console.log('HTML classes:', document.documentElement.className);
console.log('Body classes:', document.body.className);
console.log('');

// 4. 测试修复方案
console.log('=== 如果白边框还在，请运行以下代码测试 ===');
console.log('测试1: 强制移除所有margin/padding');
console.log('  document.documentElement.style.margin = "0";');
console.log('  document.documentElement.style.padding = "0";');
console.log('  document.body.style.margin = "0";');
console.log('  document.body.style.padding = "0";');
console.log('');
console.log('测试2: 强制设置背景色');
console.log('  document.documentElement.style.backgroundColor = "#000";');
console.log('  document.body.style.backgroundColor = "#000";');
