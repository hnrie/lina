#include <d3d11.h>
#include <mutex>
#include <string>
#include <vector>
#include <deque>
#include <chrono>

#include "./imdoc/imgui.cpp"
#include "./imdoc/imgui_draw.cpp"
#include "./imdoc/imgui_tables.cpp"
#include "./imdoc/imgui_widgets.cpp"
#include "./imdoc/imgui_impl_dx11.cpp"
#include "./imdoc/imgui_impl_win32.cpp"
#include "TextEditor.h"
#include "hook.hpp"

extern "C" NTSTATUS NTAPI NtProtectVirtualMemory(
    HANDLE ProcessHandle,
    PVOID *BaseAddress,
    PSIZE_T RegionSize,
    ULONG NewProtect,
    PULONG OldProtect);

extern "C" void GoDrawLoop();
extern "C" void Start();

enum NotificationType
{
    Notify_Info = 0,
    Notify_Success,
    Notify_Warning,
    Notify_Error
};

struct Notification
{
    std::string title;
    std::string message;
    NotificationType type;
    std::chrono::system_clock::time_point start_time;
    int duration_ms;
    float alpha = 0.0f;
};

static TextEditor *g_Editor = nullptr;
std::deque<Notification> g_Notifications;

void Im_RenderNotifications()
{
    const float PAD = 10.0f;
    const float WIDTH = 300.0f;
    ImGuiIO &io = ImGui::GetIO();
    float y_pos = io.DisplaySize.y - PAD;

    for (size_t i = g_Notifications.size(); i > 0;)
    {
        --i;
        Notification &n = g_Notifications[i];

        auto now = std::chrono::system_clock::now();
        auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(now - n.start_time).count();

        bool should_remove = false;

        if (elapsed > n.duration_ms)
        {
            n.alpha -= 0.05f;
            if (n.alpha <= 0.0f)
            {
                should_remove = true;
            }
        }
        else if (n.alpha < 1.0f)
        {
            n.alpha += 0.05f;
            if (n.alpha > 1.0f)
                n.alpha = 1.0f;
        }

        if (should_remove)
        {
            g_Notifications.erase(g_Notifications.begin() + i);
            continue;
        }

        ImGui::SetNextWindowBgAlpha(0.7f * n.alpha);
        ImGui::SetNextWindowPos(ImVec2(io.DisplaySize.x - PAD - WIDTH, y_pos), ImGuiCond_Always, ImVec2(0.0f, 1.0f));
        ImGui::SetNextWindowSize(ImVec2(WIDTH, 0));

        ImVec4 title_color;
        switch (n.type)
        {
        case Notify_Success:
            title_color = ImVec4(0.2f, 0.8f, 0.2f, n.alpha);
            break;
        case Notify_Warning:
            title_color = ImVec4(1.0f, 0.8f, 0.0f, n.alpha);
            break;
        case Notify_Error:
            title_color = ImVec4(1.0f, 0.2f, 0.2f, n.alpha);
            break;
        default:
            title_color = ImVec4(0.2f, 0.6f, 1.0f, n.alpha);
            break;
        }

        char window_id[32];
        sprintf(window_id, "##Notify_%zu", i);

        ImGui::PushStyleVar(ImGuiStyleVar_WindowRounding, 5.0f);
        if (ImGui::Begin(window_id, nullptr, ImGuiWindowFlags_NoDecoration | ImGuiWindowFlags_AlwaysAutoResize | ImGuiWindowFlags_NoSavedSettings | ImGuiWindowFlags_NoFocusOnAppearing | ImGuiWindowFlags_NoNav | ImGuiWindowFlags_NoInputs))
        {
            ImGui::TextColored(title_color, "%s", n.title.c_str());
            ImGui::Separator();
            ImGui::PushTextWrapPos(ImGui::GetContentRegionAvail().x);
            ImGui::TextColored(ImVec4(1, 1, 1, n.alpha), "%s", n.message.c_str());
            ImGui::PopTextWrapPos();

            y_pos -= (ImGui::GetWindowHeight() + PAD);
        }
        ImGui::End();
        ImGui::PopStyleVar();
    }
}

extern "C"
{
    void Editor_Init()
    {
        if (g_Editor)
            return;
        g_Editor = new TextEditor();
        g_Editor->SetLanguageDefinition(TextEditor::LanguageDefinition::Lua());
        g_Editor->SetPalette(TextEditor::GetDarkPalette());
    }
    void Editor_Render(char *title, float w, float h)
    {
        if (!g_Editor)
            return;
        g_Editor->Render(title, ImVec2(w, h));
    }

    void Editor_SetText(char *text)
    {
        if (!g_Editor)
            return;
        g_Editor->SetText(text);
    }
    const char *Editor_GetText()
    {
        if (!g_Editor)
            return "";
        static std::string str;
        str = g_Editor->GetText();
        return str.c_str();
    }
    void Im_PushStyleColor(int idx, float r, float g, float b, float a)
    {
        ImGui::PushStyleColor(idx, ImVec4(r, g, b, a));
    }
    void Im_PopStyleColor(int count)
    {
        ImGui::PopStyleColor(count);
    }
    void Im_Notify(char *title, char *msg, int type, int duration)
    {
        if (title == nullptr)
            title = (char *)"Notification";
        if (msg == nullptr)
            msg = (char *)"";

        Notification n;
        n.title = title;
        n.message = msg;
        n.type = (NotificationType)type;
        n.duration_ms = duration > 0 ? duration : 3000;
        n.start_time = std::chrono::system_clock::now();
        n.alpha = 0.0f;

        g_Notifications.push_back(n);
    }
    void Im_Begin(char *name, bool *open, int flags) { ImGui::Begin(name, open, flags); }
    void Im_End() { ImGui::End(); }
    void Im_Text(char *text) { ImGui::Text("%s", text); }
    bool Im_Button(char *label) { return ImGui::Button(label); }
    void Im_SetNextWindowSize(float w, float h) { ImGui::SetNextWindowSize(ImVec2(w, h)); }

    bool Im_InputTextMultiline(char *label, char *buf, size_t buf_size, float size_x, float size_y)
    {
        return ImGui::InputTextMultiline(label, buf, buf_size, ImVec2(size_x, size_y));
    }
    bool Im_BeginTabBar(char *str_id) { return ImGui::BeginTabBar(str_id); }
    void Im_EndTabBar() { ImGui::EndTabBar(); }
    bool Im_BeginTabItem(char *label) { return ImGui::BeginTabItem(label); }
    void Im_EndTabItem() { ImGui::EndTabItem(); }
    bool Im_BeginCombo(char *label, char *preview_value) { return ImGui::BeginCombo(label, preview_value); }
    void Im_EndCombo() { ImGui::EndCombo(); }
    bool Im_Selectable(char *label, bool selected) { return ImGui::Selectable(label, selected); }
    void Im_SetItemDefaultFocus() { ImGui::SetItemDefaultFocus(); }
    void Im_SameLine() { ImGui::SameLine(); }
    bool Im_Checkbox(char *label, bool *v) { return ImGui::Checkbox(label, v); }
    bool Im_SliderFloat(char *label, float *v, float v_min, float v_max, char *format)
    {
        return ImGui::SliderFloat(label, v, v_min, v_max, format);
    }
    bool Im_SliderInt(char *label, int *v, int v_min, int v_max, char *format)
    {
        return ImGui::SliderInt(label, v, v_min, v_max, format);
    }
    bool Im_ColorEdit4(char *label, float *col, int flags)
    {
        return ImGui::ColorEdit4(label, col, flags);
    }
    void Im_Separator() { ImGui::Separator(); }
    void Im_Spacing() { ImGui::Spacing(); }
    void Im_Dummy(float w, float h) { ImGui::Dummy(ImVec2(w, h)); }
    void Im_Indent(float w) { ImGui::Indent(w); }
    void Im_Unindent(float w) { ImGui::Unindent(w); }
    bool Im_CollapsingHeader(char *label, int flags) { return ImGui::CollapsingHeader(label, flags); }
    bool Im_TreeNode(char *label) { return ImGui::TreeNode(label); }
    void Im_TreePop() { ImGui::TreePop(); }
    bool Im_InputText(char *label, char *buf, size_t buf_size)
    {
        return ImGui::InputText(label, buf, buf_size);
    }

    bool Im_BeginChild(char *str_id, float w, float h, bool border, int flags)
    {
        return ImGui::BeginChild(str_id, ImVec2(w, h), border, flags);
    }
    void Im_EndChild()
    {
        ImGui::EndChild();
    }
    void Im_PushStyleVarFloat(int idx, float val)
    {
        ImGui::PushStyleVar(idx, val);
    }
    void Im_PushStyleVarVec2(int idx, float val_x, float val_y)
    {
        ImGui::PushStyleVar(idx, ImVec2(val_x, val_y));
    }
    void Im_PopStyleVar(int count)
    {
        ImGui::PopStyleVar(count);
    }
    float Im_GetWindowWidth()
    {
        return ImGui::GetWindowWidth();
    }
    float Im_GetWindowHeight()
    {
        return ImGui::GetWindowHeight();
    }
    float Im_GetContentRegionAvailWidth()
    {
        return ImGui::GetContentRegionAvail().x;
    }
    float Im_GetContentRegionAvailHeight()
    {
        return ImGui::GetContentRegionAvail().y;
    }
}

class directx_hook_t
{
public:
    static HWND window;
    static DXGI_SWAP_CHAIN_DESC swapchain_desc;
    static ID3D11Device *d3d_device;
    static ID3D11DeviceContext *device_context;
    static ID3D11RenderTargetView *render_target_view;
    static ID3D11Texture2D *back_buffer;
    static IDXGISwapChain *swap_chain;
    static bool ShouldRenderUI;

    typedef HRESULT(__stdcall *ID3D11Present)(IDXGISwapChain *SwapChain, UINT SyncInterval, UINT Flags);
    typedef HRESULT(__stdcall *ID3D11ResizeBuffers)(IDXGISwapChain *SwapChain, UINT BufferCount, UINT Width, UINT Height, DXGI_FORMAT NewFormat, UINT SwapChainFlags);
    typedef LRESULT(__stdcall *ID3D11WindowProcess)(HWND, UINT, WPARAM, LPARAM);

    static ID3D11Present original_present;
    static ID3D11ResizeBuffers original_resize_buffers;
    static ID3D11WindowProcess original_window_callback;

    static float dpi_scale;
    static bool enable_branding;

    static UINT cached_width;
    static UINT cached_height;

    static LRESULT __stdcall window_callback(HWND hwnd, UINT msg, WPARAM wparam, LPARAM lparam)
    {
        if (msg == WM_KEYDOWN)
        {
            if (wparam == VK_INSERT)
            {
                ShouldRenderUI = !ShouldRenderUI;
            }
        }
        else if (msg == WM_DPICHANGED)
        {
            dpi_scale = LOWORD(wparam) / 96.0f;
        }

        if (ImGui_ImplWin32_WndProcHandler(hwnd, msg, wparam, lparam))
        {
            return true;
        }

        switch (msg)
        {
        case WM_MOUSEWHEEL:
        case WM_LBUTTONDOWN:
        case WM_RBUTTONDOWN:
        case WM_LBUTTONUP:
        case WM_RBUTTONUP:
        case WM_KEYUP:
        case WM_CAPTURECHANGED:
        case WM_NCACTIVATE:
        case WM_CHAR:
        case WM_NCHITTEST:
        case WM_GETICON:
        case WM_INPUT:
        case WM_XBUTTONDOWN:
        case WM_XBUTTONUP:
        case WM_APPCOMMAND:
            if (ShouldRenderUI)
            {
                return true;
            }
            break;
        }

        return CallWindowProc(original_window_callback, hwnd, msg, wparam, lparam);
    }

    static void on_render_step()
    {
        static std::once_flag init_flag;
        std::call_once(init_flag, []
                       {
        if (FAILED(swap_chain->GetDesc(&swapchain_desc)))
            return;

        if (FAILED(swap_chain->GetDevice(
                __uuidof(ID3D11Device),
                (void**)&d3d_device)))
            return;

        d3d_device->GetImmediateContext(&device_context);

        if (FAILED(swap_chain->GetBuffer(
                0,
                __uuidof(ID3D11Texture2D),
                (void**)&back_buffer)))
            return;

        d3d_device->CreateRenderTargetView(
            back_buffer,
            nullptr,
            &render_target_view);

        back_buffer->Release();
        back_buffer = nullptr;

        window = swapchain_desc.OutputWindow;
        HDC hdc = GetDC(window);
        dpi_scale = GetDeviceCaps(hdc, LOGPIXELSX) / 96.0f;
        ReleaseDC(window, hdc);

        original_window_callback =
            (WNDPROC)SetWindowLongPtr(
                window,
                GWLP_WNDPROC,
                (LONG_PTR)window_callback);

        ImGui::CreateContext();
        ImGuiIO& io = ImGui::GetIO();
        io.ConfigFlags |= ImGuiConfigFlags_DockingEnable;
        io.IniFilename = nullptr;

        ImGui::StyleColorsDark();

        ImGuiStyle& s = ImGui::GetStyle();
        s.WindowRounding = 6.0f;
        s.FrameRounding  = 4.0f;
        s.ScrollbarRounding = 6.0f;
        io.FontGlobalScale = 1.0f / dpi_scale;

        ImGui_ImplWin32_Init(window);
        ImGui_ImplDX11_Init(d3d_device, device_context); });

        if (!render_target_view)
        {
            if (SUCCEEDED(swap_chain->GetBuffer(
                    0,
                    __uuidof(ID3D11Texture2D),
                    (void **)&back_buffer)))
            {
                d3d_device->CreateRenderTargetView(
                    back_buffer,
                    nullptr,
                    &render_target_view);
                back_buffer->Release();
            }
        }

        ImGui_ImplDX11_NewFrame();
        ImGui_ImplWin32_NewFrame();
        ImGui::NewFrame();

        if (enable_branding)
        {
            ImGuiIO &io = ImGui::GetIO();

            ImGui::SetNextWindowPos(
                ImVec2(10.0f, io.DisplaySize.y - 10.0f),
                ImGuiCond_Always,
                ImVec2(0.0f, 1.0f));

            ImGui::SetNextWindowBgAlpha(0.3f);

            if (ImGui::Begin(
                    "Branding",
                    nullptr,
                    ImGuiWindowFlags_NoDecoration |
                        ImGuiWindowFlags_AlwaysAutoResize |
                        ImGuiWindowFlags_NoSavedSettings |
                        ImGuiWindowFlags_NoFocusOnAppearing |
                        ImGuiWindowFlags_NoNav))
            {
                ImGui::Text("v%s %s", "1.0", ".gg/getluna");
            }
            ImGui::End();
        }

        ProcessQ();

        if (ShouldRenderUI)
        {
            GoDrawLoop();
        }

        Im_RenderNotifications();

        ImGui::EndFrame();
        ImGui::Render();

        device_context->OMSetRenderTargets(1, &render_target_view, nullptr);
        ImGui_ImplDX11_RenderDrawData(ImGui::GetDrawData());
    }
    static HRESULT __stdcall swapchain_present(IDXGISwapChain *pSwapChain, UINT sync_interval, UINT flags)
    {
        on_render_step();
        return original_present(pSwapChain, sync_interval, flags);
    }
    static HRESULT __stdcall swapchain_resize_buffers(IDXGISwapChain *swap_chain, UINT buffer_count, UINT width, UINT height, DXGI_FORMAT new_format, UINT flags)
    {
        if (render_target_view)
        {
            render_target_view->Release();
            render_target_view = nullptr;
        }
        return original_resize_buffers(swap_chain, buffer_count, width, height, new_format, flags);
    };
};

HWND directx_hook_t::window = nullptr;
DXGI_SWAP_CHAIN_DESC directx_hook_t::swapchain_desc = {};
ID3D11Device *directx_hook_t::d3d_device = nullptr;
ID3D11DeviceContext *directx_hook_t::device_context = nullptr;
ID3D11RenderTargetView *directx_hook_t::render_target_view = nullptr;
ID3D11Texture2D *directx_hook_t::back_buffer = nullptr;
IDXGISwapChain *directx_hook_t::swap_chain = nullptr;
directx_hook_t::ID3D11Present directx_hook_t::original_present = nullptr;
bool directx_hook_t::ShouldRenderUI = false;
directx_hook_t::ID3D11ResizeBuffers directx_hook_t::original_resize_buffers = nullptr;

WNDPROC directx_hook_t::original_window_callback = nullptr;
float directx_hook_t::dpi_scale = 1.0f;
bool directx_hook_t::enable_branding = true;

extern "C" void InstallCPP_Hook(uintptr_t renderJobAddr)
{
    uintptr_t viewBase = *reinterpret_cast<uintptr_t *>(renderJobAddr + 0x1d0);
    uintptr_t deviceD3D = *reinterpret_cast<uintptr_t *>(viewBase + 0x8);
    directx_hook_t::swap_chain = *reinterpret_cast<IDXGISwapChain **>(deviceD3D + 0xC8);

    uintptr_t *original_vtable = *reinterpret_cast<uintptr_t **>(directx_hook_t::swap_chain);

    uintptr_t *shadow_vtable = new uintptr_t[18];

    memcpy(shadow_vtable, original_vtable, 18 * sizeof(uintptr_t));

    directx_hook_t::original_present =
        (directx_hook_t::ID3D11Present)original_vtable[8];

    directx_hook_t::original_resize_buffers =
        (directx_hook_t::ID3D11ResizeBuffers)original_vtable[13];

    shadow_vtable[8] = (uintptr_t)directx_hook_t::swapchain_present;
    shadow_vtable[13] = (uintptr_t)directx_hook_t::swapchain_resize_buffers;
    *reinterpret_cast<uintptr_t **>(directx_hook_t::swap_chain) = shadow_vtable;
}