'use client';

interface User {
  id: string;
  email: string;
  name: string;
  picture: string;
  token?: string;
}

interface ProfileProps {
  user: User;
  onLogout: () => void;
}

export default function Profile({ user, onLogout }: ProfileProps) {
  return (
    <div className="bg-[#fffef9] rounded-[24px] shadow-sm p-6 border border-gray-100">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          {/* Avatar */}
          <div className="w-20 h-20 rounded-full overflow-hidden bg-gradient-to-br from-[#f6e58d] to-[#f9ca24] flex items-center justify-center shadow-sm">
            {user.picture ? (
              <img 
                src={user.picture} 
                alt={user.name}
                className="w-full h-full object-cover"
                referrerPolicy="no-referrer"
              />
            ) : (
              <span className="text-4xl">🍪</span>
            )}
          </div>
          
          {/* User Info */}
          <div>
            <h3 className="text-2xl font-extrabold text-gray-800">
              {user.name}
            </h3>
          </div>
        </div>

        {/* Logout Button */}
        <button
          onClick={onLogout}
          className="px-8 py-3 bg-[#f6e58d] hover:bg-[#f9ca24] text-black font-bold rounded-[24px] shadow-[0_6px_0_0_#f9ca24] hover:shadow-[0_6px_0_0_#f0932b] active:shadow-[0_2px_0_0_#f0932b] active:translate-y-[4px] transition-all duration-75"
        >
          Logout
        </button>
      </div>
    </div>
  );
}
