// 查找所有带圆角的元素
// 在侧边栏Console中运行

console.log('=== 查找所有带圆角的元素 ===\n');

const elements = document.querySelectorAll('*');
const roundedElements = [];

elements.forEach(el => {
    const styles = getComputedStyle(el);
    const borderRadius = styles.borderRadius;

    if (borderRadius && borderRadius !== '0px') {
        const rect = el.getBoundingClientRect();
        roundedElements.push({
            element: el,
            tag: el.tagName,
            class: el.className,
            borderRadius: borderRadius,
            border: styles.border,
            outline: styles.outline,
            boxShadow: styles.boxShadow,
            width: rect.width,
            height: rect.height
        });
    }
});

console.log(`找到 ${roundedElements.length} 个带圆角的元素：\n`);

roundedElements.forEach((item, index) => {
    console.log(`${index + 1}. ${item.tag}.${item.class}`);
    console.log(`   Border-radius: ${item.borderRadius}`);
    console.log(`   Border: ${item.border}`);
    console.log(`   Outline: ${item.outline}`);
    console.log(`   Box-shadow: ${item.boxShadow}`);
    console.log(`   Size: ${item.width.toFixed(0)}x${item.height.toFixed(0)}`);
    console.log('');
});

// 高亮最外层的圆角元素
const largest = roundedElements.sort((a, b) =>
    (b.width * b.height) - (a.width * a.height)
)[0];

if (largest) {
    console.log('=== 最大的圆角元素（最可能是问题源） ===');
    console.log('Tag:', largest.tag);
    console.log('Class:', largest.class);
    largest.element.style.outline = '3px solid red';
    console.log('已用红色outline标记，请查看是否就是白边框的位置');
}
