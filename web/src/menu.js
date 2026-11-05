// 侧边栏菜单与权限模块的唯一定义处
export const menuGroups = [
  {
    title: '工作台',
    items: [{ path: '/', title: '数据看板', icon: 'Odometer', module: 'dashboard' }]
  },
  {
    title: '课程内容',
    items: [
      { path: '/majors', title: '专业管理', icon: 'Collection', module: 'majors' },
      { path: '/courses', title: '课程管理', icon: 'Reading', module: 'courses' },
      { path: '/paths', title: '学习路线', icon: 'Guide', module: 'paths' }
    ]
  },
  {
    title: 'AI 知识库',
    items: [{ path: '/kb', title: '资料管理', icon: 'Files', module: 'kb' }]
  },
  {
    title: '学员',
    items: [{ path: '/learners', title: '学员与记录', icon: 'UserFilled', module: 'learners' }]
  },
  {
    title: '账号与权限',
    items: [
      { path: '/users', title: '账号管理', icon: 'User', module: 'users' },
      { path: '/roles', title: '角色权限', icon: 'Key', module: 'roles' }
    ]
  },
  {
    title: '业务数据',
    items: [
      { path: '/counters', title: '计数管理', icon: 'Tickets', module: 'counters' },
      { path: '/mini', title: '小程序管理', icon: 'Iphone', module: 'mp' }
    ]
  },
  {
    title: '系统运维',
    items: [
      { path: '/logs', title: '操作日志', icon: 'Document', module: 'logs' },
      { path: '/settings', title: '系统设置', icon: 'Setting', module: 'configs' },
      { path: '/system', title: '系统信息', icon: 'Monitor', module: 'system' }
    ]
  }
]
