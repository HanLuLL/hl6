import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SEOProvider } from "@/hooks/use-seo";
import App from "./App";
import { initializeNativeClient } from "@/lib/native-client";
import "./i18n";
import "./index.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 0,
      refetchOnWindowFocus: false,
    },
  },
});

function renderApp() {
  createRoot(document.getElementById("root")!).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <SEOProvider>
            <App />
          </SEOProvider>
          <Toaster />
        </TooltipProvider>
      </QueryClientProvider>
    </StrictMode>
  );
}

void initializeNativeClient().finally(renderApp);
