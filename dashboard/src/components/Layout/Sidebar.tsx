import { Link, useLocation } from 'react-router-dom';

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: '📊' },
  { name: 'Services', href: '/services', icon: '🔧' },
  { name: 'Recommendations', href: '/recommendations', icon: '💡' },
];

export default function Sidebar() {
  const location = useLocation();

  return (
    <div className="flex h-full w-64 flex-col bg-gray-900">
      {/* Logo */}
      <div className="flex h-16 items-center justify-center border-b border-gray-800">
        <h1 className="text-xl font-bold text-white">K8s Optimizer</h1>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 px-2 py-4">
        {navigation.map((item) => {
          const isActive = location.pathname === item.href;
          return (
            <Link
              key={item.name}
              to={item.href}
              className={`
                group flex items-center rounded-md px-3 py-2 text-sm font-medium
                ${
                  isActive
                    ? 'bg-gray-800 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                }
              `}
            >
              <span className="mr-3 text-xl">{item.icon}</span>
              {item.name}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="border-t border-gray-800 p-4">
        <div className="text-xs text-gray-400">
          <p>K8s Service Optimizer</p>
          <p className="mt-1">v1.0.0</p>
        </div>
      </div>
    </div>
  );
}
