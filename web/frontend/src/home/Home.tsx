import { BrowserRouter, Routes, Route } from "react-router";
import Dashboard from "./Dashboard";
import Settings from "./Settings";
import Users from "./Users";
import UserCreate from "./UserCreate";
import ProjectCreate from "./ProjectCreate";
import Project from "./Project";
import ProjectSettings from "./ProjectSettings";
import ProjectDataCreate from "./ProjectDataCreate";
import DataSchemaCreate from "./DataSchemaCreate";
import DataSchema from "./DataSchema";
import DataSchemaEdit from "./DataSchemaEdit";
import NotFound from "./NotFound";
import UserEdit from "./UserEdit";
import User from "./User";
import DataSchemas from "./DataSchemas";
import DataTypes from "./DataTypes";
import DataTypeCreate from "./DataTypeCreate";
import DataType from "./DataType";
import DataTypeTransforms from "./DataTypeTransforms";
import DataTypeTransformCreate from "./DataTypeTransformCreate";
import IngestionScripts from "./IngestionScripts";
import IngestionScriptCreate from "./IngestionScriptCreate";
import Profile from "./Profile";
import DataTypeEdit from "./DataTypeEdit";
import Data from "./Data";
import DataOrigins from "./DataOrigins";
import DataOriginEdit from "./DataOriginEdit";
import DataOriginCreate from "./DataOriginCreate";

export default function Home() {
    return (
        <BrowserRouter>
            <Routes>
                <Route index element={<Dashboard />} />
                <Route path="/settings" element={<Settings />} />
                <Route path="/profile" element={<Profile />} />
                <Route path="/users" element={<Users />} />
                <Route path="/user/create" element={<UserCreate />} />
                <Route path="/user/:user_id" element={<User />} />
                <Route path="/user/:user_id/edit" element={<UserEdit />} />
                <Route path="/data-schemas" element={<DataSchemas />} />
                <Route
                    path="/data-schema/create"
                    element={<DataSchemaCreate />}
                />
                <Route
                    path="/data-schema/:data_schema_id"
                    element={<DataSchema />}
                />
                <Route
                    path="/data-schema/:data_schema_id/edit"
                    element={<DataSchemaEdit />}
                />
                <Route path="/data-types" element={<DataTypes />} />
                <Route path="/data-type/create" element={<DataTypeCreate />} />
                <Route path="/data-type/:data_type_id" element={<DataType />} />
                <Route
                    path="/data-type/:data_type_id/edit"
                    element={<DataTypeEdit />}
                />
                <Route
                    path="/data-origin/create"
                    element={<DataOriginCreate />}
                />
                <Route path="/data-origins" element={<DataOrigins />} />
                <Route
                    path="/data-origin/:data_origin_id/edit"
                    element={<DataOriginEdit />}
                />
                <Route
                    path="/ingestion-scripts"
                    element={<IngestionScripts />}
                />
                <Route
                    path="/ingestion-script/create"
                    element={<IngestionScriptCreate />}
                />
                <Route
                    path="/data-type-transforms"
                    element={<DataTypeTransforms />}
                />
                <Route
                    path="/data-type-transform/create"
                    element={<DataTypeTransformCreate />}
                />
                <Route path="/project/create" element={<ProjectCreate />} />
                <Route path="/project/:project_id">
                    <Route index element={<Project />} />
                    <Route path="settings" element={<ProjectSettings />} />
                    <Route path="data/create" element={<ProjectDataCreate />} />
                </Route>
                <Route path="data/:data_id" element={<Data />} />
                <Route path="*" element={<NotFound />} />
            </Routes>
        </BrowserRouter>
    );
}
