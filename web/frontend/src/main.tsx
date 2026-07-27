import React from "react";
import { createRoot } from "react-dom/client";
import { TanStackDevtools } from "@tanstack/react-devtools";
import { ReactQueryDevtoolsPanel } from "@tanstack/react-query-devtools";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { formDevtoolsPlugin } from "@tanstack/react-form-devtools";
import "./style.css";
import App from "./App";

const container = document.getElementById("root");
const root = createRoot(container!);

root.render(
    <React.StrictMode>
        <App />
        <TanStackDevtools
            plugins={[
                // {
                //     name: "TanStack Query",
                //     render: <ReactQueryDevtoolsPanel />,
                // },
                // {
                //     name: "TanStack Router",
                //     render: <TanStackRouterDevtoolsPanel />,
                // },
                formDevtoolsPlugin(),
            ]}
        />
    </React.StrictMode>,
);
