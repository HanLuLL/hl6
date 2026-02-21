import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { LogtoProvider, type LogtoConfig, UserScope } from "@logto/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import App from "./App";
import "./i18n";
import "./index.css";

const logtoConfig: LogtoConfig = {
  endpoint: import.meta.env.VITE_LOGTO_ENDPOINT || "",
  appId: import.meta.env.VITE_LOGTO_APP_ID || "",
  resources: [import.meta.env.VITE_LOGTO_API_RESOURCE || ""],
  scopes: [UserScope.Email],
};

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
      refetchOnWindowFocus: false,
    },
  },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <LogtoProvider config={logtoConfig}>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <App />
          <Toaster />
        </TooltipProvider>
      </QueryClientProvider>
    </LogtoProvider>
  </StrictMode>
);
