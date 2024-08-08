// Emoji 功能
function show_emojis(flag){
    // 显示/隐藏 Emoji 列表
    const emojiList = document.getElementById('emoji-list-'+flag);
    const emojis = ['😀', '😂', '🤣','😅', '😎', '😐', '😑', '🤔', '🙄', '😏', '😥', '😮', '😒', '😓', '😲', '😤', '😭', '😱', '😳', '😡', '🤡', '👻', '💩', '👌', '👍', '👎', '✊', '👊', '👏', '🙏', '👀', '🤝', '💔', '💣', '🐶', '🌹', '🍻', '🎉','🔞', '❌','⭕'];
    // 创建 Emoji 列表
    emojiList.innerHTML = "";
    emojis.forEach(emoji => {
        const emojiSpan = document.createElement('span');
        emojiSpan.textContent = emoji;
        emojiSpan.addEventListener('click', () => {
            // ele_textarea.value += emoji;
            editor.insert(emoji);
        });
        emojiList.appendChild(emojiSpan);
    });
    emojiList.style.display = emojiList.style.display === 'block' ? 'none' : 'block';
}