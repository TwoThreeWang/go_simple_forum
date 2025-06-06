window.__unocss = {
    theme: {
        colors: {
            fg: "rgba(51, 51, 51,.7)",
            hover: "#5468ff",
            link: "#009ff7",
        },
    },
    shortcuts: {
        btn: "bg-#5e7ce0 px-4 py-0.5 text-white outline-0 rounded text-sm hover:bg-#7693f5 border-0 cursor-pointer",
        input: "focus:border-#5e7ce0 border rounded px-2 py-1 outline-0 text-sm border-#e5e7eb border-solid",
        aLink: "underline text-link mx-2",
        bLink: "text-link mx-1 hover:text-gray",
        'x-post-tag':'text-xs shadow border px-1.5 py-0.5 rounded b-solid b-coolGray cursor-pointer hover:opacity-80',
        tag1:'bg-white text-gray-500 dark:bg-slate-400 dark:text-white',
    },
}

function MediaChange(){
    const theme = localStorage.getItem('theme') ?? 'auto'
    if(theme === 'dark'){
        document.documentElement.classList.add('dark')
        $('[data-theme="light"]').hide()
        $('[data-theme="auto"]').hide()
    }else if(theme === 'auto'){
        $('[data-theme="night"]').hide()
        $('[data-theme="light"]').hide()
        const prefersDarkScheme = window.matchMedia('(prefers-color-scheme: dark)').matches;
        if (prefersDarkScheme){
            document.documentElement.classList.add('dark')
        }else{
            document.documentElement.classList.remove('dark')
        }
    }else{
        document.documentElement.classList.remove('dark')
        $('[data-theme="night"]').hide()
        $('[data-theme="auto"]').hide()
    }
}

$(function () {
    MediaChange()

    // 控制返回顶部按钮的显示/隐藏
    const backToTop = document.getElementById('back-to-top');
    window.addEventListener('scroll', () => {
        if (window.scrollY > 300) {
            backToTop.classList.remove('hidden');
        } else {
            backToTop.classList.add('hidden');
        }
    });

    $("input[name='result']").click(function () {
        const val = $(this).val()
        const index = $(this).data('index')
        if (val === 'reject') {
            $(`#reason-${index}`).show()
        } else {
            $(`#reason-${index}`).hide()
        }
    })

    const $sidebar = $("#sidebar");
    $("#showSidebar").click((e)=>{
        $sidebar.show().fadeIn()
        e.stopPropagation()
    })

    $(document.body).click(()=>{
      if ($sidebar.css('display') === 'block'){
          $sidebar.hide().fadeOut()
      }
    })
})



const toggleTheme = (flag)=>{
    var theme = 'auto';
    if(flag==='night'){
        theme = 'auto'
    }else if(flag === 'light'){
        theme = 'night'
    }else{
        theme = 'light'
    }
    if(theme=== 'night'){
        document.documentElement.classList.add('dark')
        localStorage.setItem('theme','dark')
        $('[data-theme="night"]').show()
        $('[data-theme="auto"]').hide()
        $('[data-theme="light"]').hide()
    }else if(theme=== 'light'){
        document.documentElement.classList.remove('dark')
        localStorage.setItem('theme','light')
        $('[data-theme="light"]').show()
        $('[data-theme="night"]').hide()
        $('[data-theme="auto"]').hide()
    }else{
        localStorage.setItem('theme','auto')
        $('[data-theme="auto"]').show()
        $('[data-theme="light"]').hide()
        $('[data-theme="night"]').hide()
        MediaChange()
    }
}