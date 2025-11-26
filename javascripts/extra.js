// 5G related extra JavaScript features
document.addEventListener('DOMContentLoaded', function() {
  // Add smooth scrolling effect
  document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
      e.preventDefault();
      document.querySelector(this.getAttribute('href')).scrollIntoView({
        behavior: 'smooth'
      });
    });
  });

  // Add page load animation
  const content = document.querySelector('.md-content');
  if (content) {
    content.style.opacity = '0';
    content.style.transform = 'translateY(20px)';
    setTimeout(() => {
      content.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
      content.style.opacity = '1';
      content.style.transform = 'translateY(0)';
    }, 100);
  }

  // 5G related interactive effects
  console.log('🚀 free-ran-ue Documentation loaded successfully!');
}); 