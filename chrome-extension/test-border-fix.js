// 测试脚本：尝试不同的方法隐藏Chrome侧边栏的白边框
// 在侧边栏Console中逐个运行，看哪个有效

console.log('=== 测试1: 使用负边距 ===');
document.body.style.margin = '-1px';
console.log('白边框消失了吗？如果没有，继续下一个测试\n');

// 如果测试1无效，撤销并尝试测试2
// document.body.style.margin = '0';

console.log('=== 测试2: 使用transform scale ===');
document.body.style.transform = 'scale(1.005)';
document.body.style.transformOrigin = 'center';
console.log('白边框消失了吗？如果没有，继续下一个测试\n');

// 如果测试2无效，撤销并尝试测试3
// document.body.style.transform = 'none';

console.log('=== 测试3: 使用box-shadow覆盖 ===');
document.body.style.boxShadow = '0 0 0 2px #000';
console.log('白边框消失了吗？如果没有，继续下一个测试\n');

// 如果测试3无效，撤销并尝试测试4
// document.body.style.boxShadow = 'none';

console.log('=== 测试4: 使用outline覆盖 ===');
document.body.style.outline = '2px solid #000';
document.body.style.outlineOffset = '-2px';
console.log('白边框消失了吗？\n');

console.log('请告诉我哪个测试让白边框消失了（如果有的话）');
