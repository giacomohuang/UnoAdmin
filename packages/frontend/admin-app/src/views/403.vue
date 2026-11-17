<template>
  <main class="grid min-h-screen place-content-center justify-items-center gap-3 bg-secondary p-6 text-center">
    <span class="text-7xl font-800 leading-none text-amber-500">403</span>
    <h1 class="m-0 text-xl font-600">没有访问权限</h1>
    <p class="m-0 text-sm text-tertiary">当前账号没有访问这个页面所需的权限，请联系管理员调整角色授权。</p>
    <RouterLink class="mt-2 inline-flex items-center gap-1.5 rounded-lg bg-brand px-4 py-2 text-sm text-white no-underline transition-colors hover:bg-brand-600" to="/dashboard">
      <span class="i-lucide:arrow-left h-4 w-4" />返回工作台
    </RouterLink>
  </main>
</template>
<router lang="json">
{
  "layout": false
}
</router>
