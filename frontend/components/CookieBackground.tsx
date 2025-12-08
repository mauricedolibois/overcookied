'use client';

import { useEffect, useState } from 'react';
import Image from 'next/image';

interface Cookie {
  id: number;
  x: number;
  y: number;
  rotation: number;
}

export default function CookieBackground() {
  const [cookies, setCookies] = useState<Cookie[]>([]);

  useEffect(() => {
    const generateCookies = () => {
      const spacing = 200; // Spacing between cookies
      const cols = Math.ceil(window.innerWidth / spacing) + 2;
      const rows = Math.ceil(window.innerHeight / spacing) + 2;
      
      const newCookies: Cookie[] = [];
      let id = 0;
      
      // Generate a repeating pattern
      for (let row = 0; row < rows; row++) {
        for (let col = 0; col < cols; col++) {
          newCookies.push({
            id: id++,
            x: col * spacing,
            y: row * spacing,
            rotation: (col + row * 123) % 360, // Pseudo-random but consistent
          });
        }
      }
      
      setCookies(newCookies);
    };

    generateCookies();
    
    // Regenerate on window resize
    window.addEventListener('resize', generateCookies);
    return () => window.removeEventListener('resize', generateCookies);
  }, []);

  return (
    <div className="fixed inset-0 overflow-hidden pointer-events-none z-0">
      <div className="animate-cookie-scroll-continuous relative">
        {cookies.map((cookie) => (
          <div
            key={cookie.id}
            className="absolute"
            style={{
              left: `${cookie.x - 200}px`,
              top: `${cookie.y - 200}px`,
              transform: `rotate(${cookie.rotation}deg)`,
            }}
          >
            <Image
              src="/cookie.svg"
              alt=""
              width={80}
              height={80}
              className="opacity-30"
            />
          </div>
        ))}
      </div>
    </div>
  );
}
