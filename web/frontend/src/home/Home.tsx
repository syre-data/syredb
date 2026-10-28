import { BrowserRouter, Routes, Route } from "react-router";
import Dashboard from "./Dashboard";
import Settings from "./Settings";
import Users from "../user/Users";
import UserCreate from "../user/UserCreate";
import ProjectCreate from "../project/ProjectCreate";
import Project from "../project/Project";
import ProjectSettings from "../project/ProjectSettings";
import ProjectDataCreate from "../project/ProjectDataCreate";
import DataSchemaCreate from "../data_schema/DataSchemaCreate";
import DataSchema from "../data_schema/DataSchema";
import DataSchemaEdit from "../data_schema/DataSchemaEdit";
import DataSchemas from "../data_schema/DataSchemas";
import NotFound from "./NotFound";
import UserEdit from "../user/UserEdit";
import User from "../user/User";
import DataTypes from "../data_type/DataTypes";
import DataTypeCreate from "../data_type/DataTypeCreate";
import DataType from "../data_type/DataType";
import DataTypeTransforms from "../transform/DataTypeTransforms";
import DataTypeTransformCreate from "../transform/DataTypeTransformCreate";
import Profile from "./Profile";
import DataTypeEdit from "../data_type/DataTypeEdit";
import Data from "./Data";
import DataEdit from "./DataEdit";
import DataOrigins from "../data_origin/DataOrigins";
import DataOriginEdit from "../data_origin/DataOriginEdit";
import DataOriginCreate from "../data_origin/DataOriginCreate";
import Logout from "./Logout";

export default function Home() {
    return (
        <BrowserRouter>
            <Routes>
                <Route index element={<Dashboard />} />
                <Route path="/logout" element={<Logout />} />
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
                <Route path="data/:data_id/edit" element={<DataEdit />} />
                <Route path="*" element={<NotFound />} />
            </Routes>
        </BrowserRouter>
    );
}
