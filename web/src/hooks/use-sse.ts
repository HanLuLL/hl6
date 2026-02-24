import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";

export function useNotificationSSE(isAuthenticated: boolean) {
  const queryClient = useQueryClient();
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!isAuthenticated) return;

    const es = new EventSource("/api/v1/notifications/sse");
    esRef.current = es;

    const handleEvent = () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
      queryClient.invalidateQueries({ queryKey: ["notifications-unread"] });
    };

    es.addEventListener("new_notification", handleEvent);
    es.addEventListener("delete_notification", handleEvent);

    es.onerror = () => {
      // EventSource will auto-reconnect
    };

    return () => {
      es.removeEventListener("new_notification", handleEvent);
      es.removeEventListener("delete_notification", handleEvent);
      es.close();
      esRef.current = null;
    };
  }, [isAuthenticated, queryClient]);
}
