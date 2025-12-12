import { BrowserRouter, Routes, Route } from "react-router";
import Dashboard from "./Dashboard";
import CreateSampleGroup from "./CreateSampleGroup";
import Settings from "./Settings";

export default function Home() {
    return (
        <BrowserRouter>
            <Routes>
                <Route index element={<Dashboard />} />
                <Route path="/settings" element={<Settings />} />
                <Route
                    path="/sample_group/create"
                    element={<CreateSampleGroup />}
                />
            </Routes>
        </BrowserRouter>
    );
}
