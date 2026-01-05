import { useEffect, useRef, useState, useCallback } from 'react';
import { UserSession } from '@/lib/auth';

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws';

export type GameState = {
    timeRemaining: number;
    p1Score: number;
    p2Score: number;
    p1Name: string;
    p2Name: string;
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

        const ws = new WebSocket(`${WS_URL}?userId=${user.id}`);

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
