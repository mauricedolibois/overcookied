import { useEffect, useRef, useState, useCallback } from 'react';
import { UserSession } from '@/lib/auth';

// WebSocket URL is computed at connection time to avoid build-time evaluation
// In development, NEXT_PUBLIC_API_URL points to localhost:8080 (backend)
// In production, it's empty so we derive from window.location
const getWsUrl = (): string => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL;
    
    // Development: Use API URL for WebSocket (localhost:8080)
    if (apiUrl) {
        const wsProtocol = apiUrl.startsWith('https') ? 'wss:' : 'ws:';
        // Extract host from API URL (e.g., "http://localhost:8080" -> "localhost:8080")
        const host = apiUrl.replace(/^https?:\/\//, '');
        return `${wsProtocol}//${host}/ws`;
    }
    
    // Production: Derive WebSocket URL from current browser location
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}/ws`;
};

export type GameState = {
    timeRemaining: number;
    p1Score: number;
    p2Score: number;
    p1Name: string;
    p2Name: string;
    p1Picture?: string;
    p2Picture?: string;
    role?: string; // 'p1' or 'p2' - indicates which player we are
    winner?: string;
    goldenCookieClaimedBy?: string;
    reason?: string;
};

export type GameMessage = {
    type: string;
    payload: any;
};

export const useGameSocket = (user: UserSession | null) => {
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const [isConnected, setIsConnected] = useState(false);
    const [gameState, setGameState] = useState<GameState | null>(null);
    const [opponentClick, setOpponentClick] = useState<{ count: number; timestamp: number } | null>(null);
    const [goldenCookieInfo, setGoldenCookieInfo] = useState<{ x: number; y: number; timestamp: number } | null>(null);
    const [gameStatus, setGameStatus] = useState<'IDLE' | 'MATCHMAKING' | 'PLAYING' | 'FINISHED'>('IDLE');
    const [powerUpExpiresAt, setPowerUpExpiresAt] = useState<number | null>(null);

    const connect = useCallback(() => {
        if (!user) return;

        const wsUrl = getWsUrl();
        // Use JWT token for secure WebSocket authentication instead of userId
        const ws = new WebSocket(`${wsUrl}?token=${encodeURIComponent(user.token)}`);

        ws.onopen = () => {
            console.log('Connected to Game Server');
            setIsConnected(true);
            // Auto join queue on connect
            ws.send(JSON.stringify({ type: 'JOIN_QUEUE', payload: {} }));
            setGameStatus('MATCHMAKING');
        };

        ws.onmessage = (event) => {
            try {
                const msg: GameMessage = JSON.parse(event.data);
                handleMessage(msg);
            } catch (e) {
                console.error('Failed to parse WS message', e);
            }
        };

        ws.onclose = () => {
            console.log('Disconnected from Game Server');
            setIsConnected(false);
        };

        setSocket(ws);

        return () => {
            ws.close();
        };
    }, [user]);

    const handleMessage = (msg: GameMessage) => {
        switch (msg.type) {
            case 'GAME_START':
                setGameStatus('PLAYING');
                // Set initial game state from GAME_START payload
                if (msg.payload.timeRemaining !== undefined) {
                    setGameState({
                        timeRemaining: msg.payload.timeRemaining,
                        p1Score: msg.payload.p1Score || 0,
                        p2Score: msg.payload.p2Score || 0,
                        p1Name: msg.payload.p1Name || '',
                        p2Name: msg.payload.p2Name || '',
                        p1Picture: msg.payload.p1Picture || '',
                        p2Picture: msg.payload.p2Picture || '',
                        role: msg.payload.role || 'p1', // Store which player we are (p1 or p2)
                    });
                }
                break;
            case 'UPDATE':
                setGameState((prev) => ({ ...prev, ...msg.payload }));
                if (msg.payload.goldenCookieClaimedBy) {
                    setGoldenCookieInfo(null); // Hide golden cookie for everyone

                    // Check if WE claimed it
                    if (msg.payload.goldenCookieClaimedBy === user?.id) {
                        setPowerUpExpiresAt(Date.now() + 5000);
                        setTimeout(() => setPowerUpExpiresAt(null), 5000);
                    }
                }
                break;
            case 'COOKIE_SPAWN':
                setGoldenCookieInfo({
                    x: msg.payload.x,
                    y: msg.payload.y,
                    timestamp: Date.now()
                });
                break;
            case 'OPPONENT_CLICK':
                setOpponentClick({
                    count: msg.payload.count,
                    timestamp: Date.now()
                });
                break;
            case 'GAME_OVER':
                setGameStatus('FINISHED');
                setGameState((prev) => {
                    // Prevent overwriting if we already have a winner (game finished)
                    if (prev?.winner) {
                        return prev;
                    }
                    return prev ? ({ ...prev, winner: msg.payload.winner, reason: msg.payload.reason }) : null;
                });
                break;
        }
    };

    const sendClick = (double: boolean = false) => {
        if (socket && isConnected) {
            socket.send(JSON.stringify({ type: 'CLICK', payload: { count: double ? 2 : 1 } }));
        }
    };

    const claimGoldenCookie = () => {
        if (socket && isConnected) {
            socket.send(JSON.stringify({ type: 'COOKIE_CLICK', payload: {} }));
            setGoldenCookieInfo(null); // Hide locally immediately
        }
    };

    const quitGame = () => {
        if (socket && isConnected) {
            socket.send(JSON.stringify({ type: 'QUIT_GAME', payload: {} }));
            setGameStatus('FINISHED');
        }
    };

    useEffect(() => {
        if (user) {
            const cleanup = connect();
            return cleanup;
        }
    }, [user, connect]);

    return {
        isConnected,
        gameState,
        gameStatus,
        opponentClick,
        goldenCookieInfo,
        powerUpExpiresAt,
        sendClick,
        claimGoldenCookie,
        quitGame
    };
};
