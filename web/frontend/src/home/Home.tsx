import { BrowserRouter, Routes, Route } from "react-router";
import Dashboard from "./Dashboard";
import SampleGroupCreate from "./SampleGroupCreate";
import Settings from "./Settings";
import Users from "./Users";
import UserCreate from "./UserCreate";
import ProjectCreate from "./ProjectCreate";
import Project from "./Project";
import ProjectSettings from "./ProjectSettings";
import ProjectSamplesCreate from "./ProjectSamplesCreate";
import DataSchemaCreate from "./DataSchemaCreate";
import DataSchema from "./DataSchema";
import SampleEdit from "./SampleEdit";
import NotFound from "./NotFound";
import UserEdit from "./UserEdit";
import User from "./User";
import DataTypes from "./DataTypes";

export default function Home() {
    return (
        <BrowserRouter>
            <Routes>
                <Route index element={<Dashboard />} />
                <Route path="/settings" element={<Settings />} />
                <Route path="/users" element={<Users />} />
                <Route path="/user/create" element={<UserCreate />} />
                <Route path="/user/:user_id" element={<User />} />
                <Route path="/user/:user_id/edit" element={<UserEdit />} />
                <Route path="/data-schemas" element={<DataSchemas />} />
                <Route path="/data-types" element={<DataTypes />} />
                <Route
                    path="/data-schema/create"
                    element={<DataSchemaCreate />}
                />
                <Route
                    path="/data-schema/:data_schema_id"
                    element={<DataSchema />}
                />
                <Route path="/project">
                    <Route path="create" element={<ProjectCreate />} />
                    <Route path=":project_id">
                        <Route index element={<Project />} />
                        <Route path="settings" element={<ProjectSettings />} />
                        <Route
                            path="samples/create"
                            element={<ProjectSamplesCreate />}
                        />
                        <Route
                            path="sample/:sample_id/edit"
                            element={<SampleEdit />}
                        />
                        <Route
                            path="sample-group/create"
                            element={<SampleGroupCreate />}
                        />
                    </Route>
                </Route>
                <Route path="*" element={<NotFound />} />
            </Routes>
        </BrowserRouter>
    );
}
