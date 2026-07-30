import { createContext, useContext, useEffect, useMemo, useState } from 'react';
import { supabase } from '../lib/supabase';
import { apiGet, apiGetStrict, endBackendSession, establishBackendSession } from '../services/api';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState({ authenticated: false, loading: true });

  useEffect(() => {
    let active = true;
    async function syncUser(session) {
      if (!session) {
        if (active) setUser({ authenticated: false, loading: false });
        return;
      }
      try {
        await establishBackendSession(session.access_token);
        const profile = await apiGet('/me');
        if (active) {
          setUser(profile
            ? { ...profile, authenticated: true, loading: false }
            : { authenticated: false, loading: false });
        }
      } catch {
        if (active) setUser({ authenticated: false, loading: false });
      }
    }

    supabase.auth.getSession().then(({ data }) => syncUser(data.session));
    const { data: listener } = supabase.auth.onAuthStateChange((_event, session) => {
      window.setTimeout(() => syncUser(session), 0);
    });
    return () => {
      active = false;
      listener.subscription.unsubscribe();
    };
  }, []);

  const value = useMemo(() => ({
    user,
    async signIn(email, password) {
      const { data, error } = await supabase.auth.signInWithPassword({ email, password });
      if (error) throw error;
      try {
        await establishBackendSession(data.session.access_token);
        const profile = await apiGetStrict('/me');
        setUser({ ...profile, authenticated: true, loading: false });
      } catch (profileError) {
        await supabase.auth.signOut();
        throw new Error(`Unable to load your profile: ${profileError.message}`, { cause: profileError });
      }
    },
    async signOut() {
      try {
        await endBackendSession();
      } catch {
        // Supabase logout still completes if the backend is temporarily unavailable.
      }
      await supabase.auth.signOut();
      setUser({ authenticated: false, loading: false });
    },
  }), [user]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuthContext() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuthContext must be used within AuthProvider');
  return context;
}
